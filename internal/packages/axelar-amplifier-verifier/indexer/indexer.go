package indexer

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"

	"github.com/cosmostation/cvms/internal/common"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/packages/axelar-amplifier-verifier/repository"
)

var (
	subsystem                 = "axelar_amplifier_verifier"
	_         common.IIndexer = (*AxelarAmplifierVerifierIndexer)(nil)
)

type AxelarAmplifierVerifierIndexer struct {
	*common.Indexer
	repository.AmplifierIndexerRepository
}

func NewAxelarAmplifierVerifierIndexer(p common.Packager) (*AxelarAmplifierVerifierIndexer, error) {
	status := helper.GetOnChainStatus(p.RPCs, p.ProtocolType)
	if status.ChainID == "" {
		return nil, errors.New("failed to create new veindexer")
	}
	indexer := common.NewIndexer(p, p.Package, status.ChainID)
	repo := repository.NewRepository(*p.IndexerDB, subsystem, indexertypes.SQLQueryMaxDuration)
	indexer.Lh = indexertypes.LatestHeightCache{LatestHeight: status.BlockHeight}
	return &AxelarAmplifierVerifierIndexer{indexer, repo}, nil
}

func (idx *AxelarAmplifierVerifierIndexer) Start() error {
	err := idx.InitChainInfoID()
	if err != nil {
		return errors.Wrap(err, "failed to init chain_info_id")
	}

	alreadyInit, err := idx.CheckIndexPointerAlreadyInitialized(idx.IndexName, idx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to check init tables")
	}
	if !alreadyInit {
		idx.Warnf("it's not initialized in the database, so that this package will be init at %d", idx.Lh.LatestHeight)
		idx.InitPartitionTablesByChainInfoID(idx.IndexName, idx.ChainID, idx.Lh.LatestHeight)
		idx.CreateVerifierInfoPartitionTableByChainID(idx.ChainID)
	}

	// get last index pointer, index pointer is always initalize if not exist
	initIndexPointer, err := idx.GetLastIndexPointerByIndexTableName(idx.IndexName, idx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to get last index pointer")
	}

	err = idx.FetchValidatorInfoList()
	if err != nil {
		return errors.Wrap(err, "failed to fetch validator_info list")
	}

	idx.Infof("loaded index pointer: %d, loaded vim length: %d VAM: %d", initIndexPointer.Pointer, len(idx.Vim), len(idx.VAM))

	// init indexer metrics
	idx.initLabelsAndMetrics()
	// go fetch new height in loop, it must be after init metrics
	go idx.FetchLatestHeight()
	go idx.Loop(initIndexPointer.Pointer)
	go func() {
		for {
			idx.Infof("for time retention, delete old records over %s and sleep %s", idx.RetentionPeriod, indexertypes.RetentionQuerySleepDuration)
			idx.DeleteOldValidatorExtensionVoteList(idx.ChainID, idx.RetentionPeriod)
			time.Sleep(indexertypes.RetentionQuerySleepDuration)
		}
	}()
	return nil
}

func (idx *AxelarAmplifierVerifierIndexer) Loop(indexPoint int64) {
	isUnhealth := false
	for {
		// node health check
		if isUnhealth {
			healthAPIs := healthcheck.FilterHealthEndpoints(idx.APIs, idx.ProtocolType)
			for _, api := range healthAPIs {
				idx.SetAPIEndPoint(api)
				idx.Warnf("API endpoint will be changed with health endpoint for this package: %s", api)
				isUnhealth = false
				break
			}

			healthRPCs := healthcheck.FilterHealthRPCEndpoints(idx.RPCs, idx.ProtocolType)
			for _, rpc := range healthRPCs {
				idx.SetRPCEndPoint(rpc)
				idx.Warnf("RPC endpoint will be changed with health endpoint for this package: %s", rpc)
				isUnhealth = false
				break
			}

			if len(healthAPIs) == 0 || len(healthRPCs) == 0 {
				isUnhealth = true
				idx.Errorln("failed to get any health endpoints from healthcheck filter, retry sleep 10s")
				time.Sleep(indexertypes.UnHealthSleep)
				continue
			}
		}

		// trying to sync with new index pointer height
		newIndexPointer, err := idx.batchSync(indexPoint)
		if err != nil {
			common.Health.With(idx.RootLabels).Set(0)
			common.Ops.With(idx.RootLabels).Inc()
			isUnhealth = true
			idx.Errorf("failed to sync validators vote status in %d height: %s\nit will be retried after sleep %s...",
				indexPoint, err, indexertypes.AfterFailedRetryTimeout.String(),
			)
			time.Sleep(indexertypes.AfterFailedRetryTimeout)
			continue
		}

		// update index point
		indexPoint = newIndexPointer

		// update health and ops
		common.Health.With(idx.RootLabels).Set(1)
		common.Ops.With(idx.RootLabels).Inc()

		// logging & sleep
		if idx.Lh.LatestHeight > indexPoint {
			// when node catching_up is true, sleep 100 milli sec
			idx.WithField("catching_up", true).
				Infof("updated index pointer is %d ... remaining %d blocks", indexPoint, (idx.Lh.LatestHeight - indexPoint))
			time.Sleep(indexertypes.CatchingUpSleepDuration)
		} else {
			// when node already catched up, sleep 5 sec
			idx.WithField("catching_up", false).
				Infof("updated index pointer to %d and sleep %s sec...", indexPoint, indexertypes.DefaultSleepDuration.String())
			time.Sleep(indexertypes.DefaultSleepDuration)
		}
	}
}

// insert chain-info into chain_info table
func (idx *AxelarAmplifierVerifierIndexer) InitChainInfoID() error {
	isNewChain := false
	var chainInfoID int64
	chainInfoID, err := idx.SelectChainInfoIDByChainID(idx.ChainID)
	if err != nil {
		if err == sql.ErrNoRows {
			idx.Infof("this is new chain id: %s", idx.ChainID)
			isNewChain = true
		} else {
			return errors.Wrap(err, "failed to select chain_info_id by chain-id")
		}
	}

	if isNewChain {
		chainInfoID, err = idx.InsertChainInfo(idx.ChainName, idx.ChainID, idx.Mainnet)
		if err != nil {
			return errors.Wrap(err, "failed to insert new chain_info_id by chain-id")
		}
	}

	idx.ChainInfoID = chainInfoID
	idx.Debugf("set chain info id: %d", chainInfoID)
	return nil
}

func (idx *AxelarAmplifierVerifierIndexer) FetchValidatorInfoList() error {
	verifierInfoList, err := idx.GetVerifierInfoListByChainInfoID(idx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to get validator info list")
	}
	for _, verifier := range verifierInfoList {
		idx.Vim[verifier.VerifierAddress] = verifier.ID
		idx.VAM[verifier.ID] = verifier.VerifierAddress
	}
	return nil
}
