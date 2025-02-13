package common

import (
	"time"

	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metrics name for indexer
const (
	IndexPointerEpochMetricName          = "latest_index_pointer_epoch"
	IndexPointerBlockHeightMetricName    = "latest_index_pointer_block_height"
	IndexPointerBlockTimestampMetricName = "latest_index_pointer_block_timestamp"
	LatestBlockHeightMetricName          = "latest_block_height"
	RecentMissCounterMetricName          = "recent_miss_counter"
	CovenantSigCountMetricName           = "covenant_sigs_count"
	BtcDelegationCountTotalMetricName    = "btc_delegation_count_total"
)

type Indexer struct {
	CommonApp
	ChainName    string
	Mainnet      bool
	ChainID      string
	ChainInfoID  int64
	ProtocolType string
	IsConsumer   bool
	Monikers     []string
	MonikerIDMap indexertypes.MonikerIDMap
	Endpoints
	*IndexerDB
	Vim                indexertypes.ValidatorIDMap
	Lh                 indexertypes.LatestHeightCache
	Factory            promauto.Factory
	MetricsMap         map[string]prometheus.Gauge
	MetricsVecMap      map[string]*prometheus.GaugeVec
	MetricsCountMap    map[string]prometheus.Counter
	MetricsCountVecMap map[string]*prometheus.CounterVec
	RootLabels         prometheus.Labels
	PackageLabels      prometheus.Labels
}

// TODO: not implemented
func NewIndexer(p Packager, subsystem string, chainID string) *Indexer {
	app := NewCommonApp(p)
	// NOTE: empty monikers mean all mode
	monikers := []string{}
	if p.Mode == VALIDATOR {
		monikers = p.Monikers
	}

	// setup endpoints
	for _, rpc := range p.RPCs {
		app.SetRPCEndPoint(rpc)
		break
	}
	for _, api := range p.APIs {
		app.SetAPIEndPoint(api)
		break
	}
	if p.IsConsumerChain {
		app.OptionalClient = NewOptionalClient(app.Entry)
		for _, rpc := range p.ProviderEndPoints.RPCs {
			app.OptionalClient.SetRPCEndPoint(rpc)
			break
		}
		for _, api := range p.ProviderEndPoints.APIs {
			app.OptionalClient.SetAPIEndPoint(api)
			break
		}
	}

	return &Indexer{
		CommonApp: app,
		ChainName: p.ChainName,
		Mainnet:   p.Mainnet,
		ChainID:   p.ChainID,
		// skip chainInfoID
		ProtocolType: p.ProtocolType,
		IsConsumer:   p.IsConsumerChain,
		Monikers:     monikers,
		Endpoints:    p.Endpoints,
		IndexerDB:    p.IndexerDB,
		Vim:          make(indexertypes.ValidatorIDMap, 0),
		// skip latestHeightCache
		Factory:            p.Factory,
		MetricsMap:         map[string]prometheus.Gauge{},
		MetricsCountMap:    map[string]prometheus.Counter{},
		MetricsVecMap:      map[string]*prometheus.GaugeVec{},
		MetricsCountVecMap: map[string]*prometheus.CounterVec{},
		RootLabels:         BuildRootLabels(p),
		PackageLabels:      BuildPackageLabels(p),
	}
}

func (indexer *Indexer) FetchLatestHeight() {
	for {
		err := func() error {
			status := helper.GetOnChainStatus(indexer.RPCs, indexer.ProtocolType)

			indexer.Lh.Mutex.RLock()
			defer indexer.Lh.Mutex.RUnlock()
			indexer.Lh.LatestHeight = status.BlockHeight

			indexer.Debugf("fetched latest block height: %d and sleep %s sec...", status.BlockHeight, indexertypes.FetchSleepDuration.String())
			time.Sleep(indexertypes.FetchSleepDuration)
			return nil
		}()

		if err != nil {
			indexer.Errorf("failed to fetch height: %s and sleep %s sec...", err, indexertypes.AfterFailedFetchSleepDuration.String())
			time.Sleep(indexertypes.AfterFailedFetchSleepDuration)
		}

		// if loop is true, update metrics
		indexer.MetricsMap[LatestBlockHeightMetricName].Set(float64(indexer.Lh.LatestHeight))
		indexer.Infof("update prometheus metrics %d height", indexer.Lh.LatestHeight)
	}
}
