package gnolandvoteindexer

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/db"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"

	"github.com/cosmostation/cvms/internal/common"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/packages/consensus/voteindexer/repository"
)

var (
	subsystem = "consensus_vote"
)

type VoteIndexer struct {
	*common.Indexer
	repo repository.VoteIndexerRepository
}

// Compile-time Assertion
var _ common.IIndexer = (*VoteIndexer)(nil)

func NewVoteIndexer(p common.Packager) (*VoteIndexer, error) {
	status := helper.GetOnChainStatus(p.RPCs, p.ProtocolType)
	if status.ChainID == "" {
		return nil, errors.Errorf("failed to create new voteindexer by failing getting onchain status through %v", p.RPCs)
	}
	indexer := common.NewIndexer(p, p.Package, status.ChainID)
	repo := repository.NewRepository(*p.IndexerDB, indexertypes.SQLQueryMaxDuration)
	indexer.Lh = indexertypes.LatestHeightCache{LatestHeight: status.BlockHeight}
	return &VoteIndexer{indexer, repo}, nil
}

func (vidx *VoteIndexer) Start() error {
	err := vidx.InitChainInfoID()
	if err != nil {
		return errors.Wrap(err, "failed to init chain_info_id")
	}

	alreadyInit, err := vidx.repo.CheckIndexPointerAlreadyInitialized(repository.IndexName, vidx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to check init tables")
	}
	if !alreadyInit {
		vidx.Warnln("it's not initialized in the database, so that voteindexer will init for this package")
		vidx.repo.InitPartitionTablesByChainInfoID(repository.IndexName, vidx.ChainID, vidx.Lh.LatestHeight)
	}

	// vidx.repo.InitPartitionTablesByChainInfoID(repository.IndexName, vidx.ChainID, vidx.Lh.LatestHeight)

	// get last index pointer, index pointer is always initialize if not exist
	initIndexPointer, err := vidx.repo.GetLastIndexPointerByIndexTableName(repository.IndexName, vidx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to get last index pointer")
	}

	err = vidx.FetchValidatorInfoList()
	if err != nil {
		return errors.Wrap(err, "failed to fetch validator_info list")
	}

	vidx.Infof("loaded index pointer(last saved height): %d", initIndexPointer.Pointer)
	vidx.Infof("initial vim length: %d for %s chain", len(vidx.Vim), vidx.ChainID)

	// init indexer metrics
	vidx.initLabelsAndMetrics()
	// go fetch new height in loop, it must be after init metrics
	go vidx.FetchLatestHeight()
	// loop
	go vidx.Loop(initIndexPointer.Pointer)
	// loop update recent miss counter metrics
	go func() {
		for {
			vidx.Debugln("update recent miss counter metrics and sleep 5s sec...")
			vidx.updateRecentMissCounterMetric()
			time.Sleep(time.Second * 5)
		}
	}()
	// loop partion table time retention by env parameter
	go func() {
		if vidx.RetentionPeriod == db.PersistenceMode {
			vidx.Infoln("skipped the postgres time retention")
			return
		}
		for {
			vidx.Infof("for time retention, delete old records over %s and sleep %s", vidx.RetentionPeriod, indexertypes.RetentionQuerySleepDuration)
			vidx.repo.DeleteOldValidatorVoteList(vidx.ChainID, vidx.RetentionPeriod)
			time.Sleep(indexertypes.RetentionQuerySleepDuration)
		}
	}()
	return nil
}

func (vidx *VoteIndexer) Loop(indexPoint int64) {
	isUnhealth := false
	for {
		// node health check
		if isUnhealth {
			healthAPIs := healthcheck.FilterHealthEndpoints(vidx.APIs, vidx.ProtocolType)
			for _, api := range healthAPIs {
				vidx.SetAPIEndPoint(api)
				vidx.Warnf("API endpoint will be changed with health endpoint for this package: %s", api)
				isUnhealth = false
				break
			}

			healthRPCs := healthcheck.FilterHealthRPCEndpoints(vidx.RPCs, vidx.ProtocolType)
			for _, rpc := range healthRPCs {
				vidx.SetRPCEndPoint(rpc)
				vidx.Warnf("RPC endpoint will be changed with health endpoint for this package: %s", rpc)
				isUnhealth = false
				break
			}

			if len(healthAPIs) == 0 || len(healthRPCs) == 0 {
				isUnhealth = true
				vidx.Errorln("failed to get any health endpoints from healthcheck filter, retry sleep 10s")
				time.Sleep(indexertypes.UnHealthSleep)
				continue
			}
		}

		// set new index point height
		newIndexPointerHeight := indexPoint + 1

		// trying to sync with new index pointer height
		newIndexPointer, err := vidx.batchSync(indexPoint, newIndexPointerHeight)
		if err != nil {
			common.Health.With(vidx.RootLabels).Set(0)
			common.Ops.With(vidx.RootLabels).Inc()
			isUnhealth = true
			vidx.Errorf("failed to sync validators vote status in %d height: %s\nit will be retried after sleep %s...", indexPoint, err, indexertypes.AfterFailedFetchSleepDuration.String())
			time.Sleep(indexertypes.AfterFailedRetryTimeout)
			continue
		}

		// update index point
		indexPoint = newIndexPointer

		// update health and ops
		common.Health.With(vidx.RootLabels).Set(1)
		common.Ops.With(vidx.RootLabels).Inc()

		// logging & sleep
		if vidx.Lh.LatestHeight > indexPoint {
			// when node catching_up is true, sleep 100 milli sec
			vidx.WithField("catching_up", true).
				Infof("latest height is %d but updated index pointer is %d ... remaining %d blocks", vidx.Lh.LatestHeight, indexPoint, (vidx.Lh.LatestHeight - indexPoint))
			time.Sleep(indexertypes.CatchingUpSleepDuration)
		} else {
			// when node already caught up, sleep 5 sec
			vidx.WithField("catching_up", false).
				Infof("updated index pointer to %d and sleep %s sec...", indexPoint, indexertypes.DefaultSleepDuration.String())
			time.Sleep(indexertypes.DefaultSleepDuration)
		}
	}
}

// TODO: move into metarepo
// insert chain-info into chain_info table
func (vidx *VoteIndexer) InitChainInfoID() error {
	isNewChain := false
	var chainInfoID int64
	chainInfoID, err := vidx.repo.SelectChainInfoIDByChainID(vidx.ChainID)
	if err != nil {
		if err == sql.ErrNoRows {
			vidx.Infof("this is new chain id: %s", vidx.ChainID)
			isNewChain = true
		} else {
			return errors.Wrap(err, "failed to select chain_info_id by chain-id")
		}
	}

	if isNewChain {
		chainInfoID, err = vidx.repo.InsertChainInfo(vidx.ChainName, vidx.ChainID, vidx.Mainnet)
		if err != nil {
			return errors.Wrap(err, "failed to insert new chain_info_id by chain-id")
		}
	}

	vidx.ChainInfoID = chainInfoID
	return nil
}

func (vidx *VoteIndexer) FetchValidatorInfoList() error {
	// get already saved validator-set list for mapping validators ids
	validatorInfoList, err := vidx.repo.GetValidatorInfoListByChainInfoID(vidx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to get validator info list")
	}

	// when the this package starts, set validator-id map
	for _, validator := range validatorInfoList {
		vidx.Vim[validator.HexAddress] = int64(validator.ID)
	}

	return nil
}
