package indexer

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	commonapi "github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/common/indexer/model"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/db"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"
	"github.com/cosmostation/cvms/internal/packages/babylon/covenant-committee/repository"
	"github.com/pkg/errors"
)

var subsystem = "babylon_committee"

type CovenantSignatureIndexer struct {
	*common.Indexer
	csRepo               repository.CovenantSignatureRepository
	btcDelRepo           repository.BtcDelegationRepository
	earliestBlockHeight  int64
	covenantCommitteeMap map[string]int64
}

var _ common.IIndexer = (*CovenantSignatureIndexer)(nil)

func NewCovenantSignatureIndexer(p common.Packager) (*CovenantSignatureIndexer, error) {
	status := helper.GetOnChainStatus(p.RPCs, p.ProtocolType)
	if status.ChainID == "" {
		return nil, errors.Errorf("failed to create a new covenant signature indexer: %v", status)
	}

	indexer := common.NewIndexer(p, p.Package, status.ChainID)

	csRepo := repository.NewCovenantSigRepository(*p.IndexerDB, indexertypes.SQLQueryMaxDuration)
	btcDelRepo := repository.NewBtcDelegationRepository(*p.IndexerDB, indexertypes.SQLQueryMaxDuration)
	indexer.Lh = indexertypes.LatestHeightCache{LatestHeight: status.BlockHeight}
	return &CovenantSignatureIndexer{indexer, csRepo, btcDelRepo, status.EarliestBlockHeight, make(map[string]int64, 0)}, nil
}

func (idx *CovenantSignatureIndexer) Start() error {
	err := idx.InitChainInfoID()
	if err != nil {
		return errors.Wrap(err, "failed to init chain_info_id")
	}

	alreadyInit, err := idx.csRepo.CheckIndexPointerAlreadyInitialized(repository.IndexName, idx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to check init tables")
	}
	if !alreadyInit {
		idx.Warnf("it's not initialized in the database, so that this package will initalize at %d as a init index point", idx.Lh.LatestHeight)
		idx.Warnf("%s,%s,%d", repository.IndexName, idx.ChainID, idx.Lh.LatestHeight)
		idx.csRepo.InitPartitionTablesByChainInfoID(repository.IndexName, idx.ChainID, idx.earliestBlockHeight)
		idx.btcDelRepo.CreatePartitionTable(repository.SubIndexName, idx.ChainID)
		idx.csRepo.CreateCovenantCommitteeInfoPartitionTableByChainID(idx.ChainID)
	}

	// get last index pointer, index pointer is always initalize if not exist
	initIndexPointer, err := idx.csRepo.GetLastIndexPointerByIndexTableName(repository.IndexName, idx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to get last index pointer")
	}

	// initialize babylon covenant committee
	newCovenantCommitteeInfoList := []model.CovenantCommitteeInfo{}
	covenantCommittee, err := commonapi.GetBalbylonCovenantCommiteeParams(idx.CommonClient)
	if err != nil {
		return errors.Wrap(err, "failed to get covenant committee params")
	}
	// request covenant mapping name
	remoteCCNameList, err := testGetCovenantComitteeName()
	if err != nil {
		return errors.Wrap(err, "failed to get covenant committee params")
	}

	for _, pk := range covenantCommittee {
		comitteeName := ""
		for _, c := range remoteCCNameList.CovenantCommittee {
			if c.Pk == pk {
				comitteeName = c.Name
				break
			}
		}

		if comitteeName == "" {
			comitteeName = "Unknown"
		}

		newCovenantCommitteeInfoList = append(newCovenantCommitteeInfoList, model.CovenantCommitteeInfo{
			ChainInfoID:   idx.ChainInfoID,
			CovenantBtcPk: pk,
			Name:          comitteeName,
		})
	}

	idx.csRepo.UpsertCovenantCommitteeInfoList(newCovenantCommitteeInfoList)
	err = idx.FetchValidatorInfoList()
	if err != nil {
		return errors.Wrap(err, "failed to fetch covenant committee list")
	}

	idx.Infof("loaded last index pointer: %d", initIndexPointer.Pointer)

	// init indexer metrics
	idx.initLabelsAndMetrics()
	idx.initMetricState(idx.covenantCommitteeMap)

	// go fetch new height in loop, it must be after init metrics
	go idx.FetchLatestHeight()

	// loop
	go idx.Loop(initIndexPointer.Pointer)

	// loop partion table time retention by env parameter
	go func() {
		if idx.RetentionPeriod == db.PersistenceMode {
			idx.Infoln("skipped the postgres time retention")
			return
		}
		for {
			idx.Infof("for time retention, delete old records over %s and sleep %s", idx.RetentionPeriod, indexertypes.RetentionQuerySleepDuration)
			idx.csRepo.DeleteOldCovenantSignatureList(idx.ChainID, idx.RetentionPeriod)
			idx.btcDelRepo.DeleteOldBtcDelegationList(idx.ChainID, idx.RetentionPeriod)
			time.Sleep(indexertypes.RetentionQuerySleepDuration)
		}
	}()

	return nil
}

// insert chain-info into chain_info table
func (idx *CovenantSignatureIndexer) InitChainInfoID() error {
	isNewChain := false
	var chainInfoID int64
	chainInfoID, err := idx.csRepo.SelectChainInfoIDByChainID(idx.ChainID)
	if err != nil {
		if err == sql.ErrNoRows {
			idx.Infof("this is new chain id: %s", idx.ChainID)
			isNewChain = true
		} else {
			return errors.Wrap(err, "failed to select chain_info_id by chain-id")
		}
	}

	if isNewChain {
		chainInfoID, err = idx.csRepo.InsertChainInfo(idx.ChainName, idx.ChainID, idx.Mainnet)
		if err != nil {
			return errors.Wrap(err, "failed to insert new chain_info_id by chain-id")
		}
	}

	idx.ChainInfoID = chainInfoID
	return nil
}

func (idx *CovenantSignatureIndexer) Loop(indexPoint int64) {
	isUnhealth := false
	for {
		// node health check
		if isUnhealth {
			healthAPIs := healthcheck.FilterHealthEndpoints(idx.APIs, idx.ProtocolType)
			for _, api := range healthAPIs {
				idx.SetAPIEndPoint(api)
				idx.Debugf("API endpoint will be changed with health endpoint for this package: %s", api)
				isUnhealth = false
				break
			}

			healthRPCs := healthcheck.FilterHealthRPCEndpoints(idx.RPCs, idx.ProtocolType)
			for _, rpc := range healthRPCs {
				idx.SetRPCEndPoint(rpc)
				idx.Debugf("RPC endpoint will be changed with health endpoint for this package: %s", rpc)
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
		newIndexPointerHeight := indexPoint + 1

		// trying to sync with new index pointer height
		newIndexPointer, err := idx.batchSync(indexPoint, newIndexPointerHeight)
		if err != nil {
			common.Health.With(idx.RootLabels).Set(0)
			common.Ops.With(idx.RootLabels).Inc()
			isUnhealth = true
			idx.Errorf("failed to sync status in %d block %s, it will be retried after sleep %s...",
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
				Infof("latest height is %d but updated index pointer is %d ... remaining %d blocks", idx.Lh.LatestHeight, indexPoint, (idx.Lh.LatestHeight - indexPoint))
			time.Sleep(indexertypes.CatchingUpSleepDuration)
		} else {
			// when node already catched up, sleep 5 sec
			idx.WithField("catching_up", false).
				Infof("updated index pointer to %d and sleep %s sec...", indexPoint, indexertypes.DefaultSleepDuration.String())
			time.Sleep(indexertypes.DefaultSleepDuration)
		}
	}
}

func (idx *CovenantSignatureIndexer) FetchValidatorInfoList() error {
	// get already saved covenant committee members list for mapping covenant committee ids
	covenantCommitteeInfoList, err := idx.csRepo.GetCovenantCommitteeInfoListByChainInfoID(idx.ChainInfoID)
	if err != nil {
		return errors.Wrap(err, "failed to get validator info list")
	}

	// when the this pacakge starts, set committee-id map
	for _, committee := range covenantCommitteeInfoList {
		idx.covenantCommitteeMap[committee.CovenantBtcPk] = int64(committee.ID)
	}

	return nil
}

func testGetCovenantComitteeName() (CovenantCommitteeListFromRemoteRepo, error) {
	rawJson := []byte(`{
    "covenant_committee": [
        {
        "name": "Babylon Labs",
        "pk": "fa9d882d45f4060bdb8042183828cd87544f1ea997380e586cab77d5fd698737"
        },
        {
        "name": "Babylon Labs",
        "pk": "0aee0509b16db71c999238a4827db945526859b13c95487ab46725357c9a9f25"
        },
        {
        "name": "Babylon Labs",
        "pk": "17921cf156ccb4e73d428f996ed11b245313e37e27c978ac4d2cc21eca4672e4"
        },
        {
        "name": "Cubist",
        "pk": "113c3a32a9d320b72190a04a020a0db3976ef36972673258e9a38a364f3dc3b0"
        },
        {
        "name": "Informal Systems",
        "pk": "79a71ffd71c503ef2e2f91bccfc8fcda7946f4653cef0d9f3dde20795ef3b9f0"
        },
        {
        "name": "Zellic",
        "pk": "3bb93dfc8b61887d771f3630e9a63e97cbafcfcc78556a474df83a31a0ef899c"
        },
        {
        "name": "RockX",
        "pk": "d21faf78c6751a0d38e6bd8028b907ff07e9a869a43fc837d6b3f8dff6119a36"
        },
        {
        "name": "AltLayer",
        "pk": "f5199efae3f28bb82476163a7e458c7ad445d9bffb0682d10d3bdb2cb41f8e8e"
        },
        {
        "name": "CoinSummer Labs",
        "pk": "40afaf47c4ffa56de86410d8e47baa2bb6f04b604f4ea24323737ddc3fe092df"
        }
    ]
}`)
	var covenantCommiteeNameList CovenantCommitteeListFromRemoteRepo

	err := json.Unmarshal(rawJson, &covenantCommiteeNameList)
	if err != nil {
		return CovenantCommitteeListFromRemoteRepo{}, err
	}

	return covenantCommiteeNameList, nil
}
