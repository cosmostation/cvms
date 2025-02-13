package indexer

import (
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-covenant-signature/model"
	"github.com/prometheus/client_golang/prometheus"
)

func (idx *CovenantSignatureIndexer) initLabelsAndMetrics() {
	covenantSigMetric := idx.Factory.NewCounterVec(prometheus.CounterOpts{
		Namespace:   common.Namespace,
		Subsystem:   subsystem,
		Name:        common.CovenantSigCountMetricName,
		ConstLabels: idx.PackageLabels,
	}, []string{
		"btc_pk",
	})
	idx.MetricsCountVecMap[common.CovenantSigCountMetricName] = covenantSigMetric

	findBtcDelegationMetric := idx.Factory.NewCounter(prometheus.CounterOpts{
		Namespace: common.Namespace,
		Subsystem: subsystem,
		Name:      common.BtcDelegationCountTotalMetricName,
		Help:      "The total number of BTC delegations found at the time the indexer started.",
	})

	idx.MetricsCountMap[common.BtcDelegationCountTotalMetricName] = findBtcDelegationMetric

	latestBlockHeightMetric := idx.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   subsystem,
		Name:        common.LatestBlockHeightMetricName,
		ConstLabels: idx.PackageLabels,
	})

	latestBlockHeightMetric.Set(0)
	idx.MetricsMap[common.LatestBlockHeightMetricName] = latestBlockHeightMetric

	timestampMetric := idx.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   subsystem,
		Name:        common.IndexPointerBlockTimestampMetricName,
		ConstLabels: idx.PackageLabels,
	})
	idx.MetricsMap[common.IndexPointerBlockTimestampMetricName] = timestampMetric
}

func (idx *CovenantSignatureIndexer) initMetricState(covenantCommitteeMap map[string]int64) {
	for btcPk, _ := range covenantCommitteeMap {
		covenantSigMetric, ok := idx.MetricsCountVecMap[common.CovenantSigCountMetricName]
		if ok {
			covenantSigMetric.WithLabelValues(btcPk).Add(0)
		}
	}

	btcDelegationMetrics, ok := idx.MetricsCountMap[common.BtcDelegationCountTotalMetricName]
	if ok {
		btcDelegationMetrics.Add(0)
	}
}

func (idx *CovenantSignatureIndexer) updatePrometheusMetrics(
	covenantSignatureList []model.BabylonCovenantSignature,
	btcDelegationsList []model.BabylonBtcDelegation,
	indexPointerTimestamp time.Time,
) {
	covenantSigMetric, ok := idx.MetricsCountVecMap[common.CovenantSigCountMetricName]
	if ok {
		for _, sig := range covenantSignatureList {
			for btcPk, id := range idx.covenantCommitteeMap {
				if id == sig.CovenantBtcPkID {
					// add count
					covenantSigMetric.WithLabelValues(btcPk).Add(1)
				}
			}
		}
	}

	btcDelegationMetrics, ok := idx.MetricsCountMap[common.BtcDelegationCountTotalMetricName]
	if ok {
		btcDelegationMetrics.Add(float64(len(btcDelegationsList)))
	}
	idx.MetricsMap[common.IndexPointerBlockTimestampMetricName].Set((float64(indexPointerTimestamp.Unix())))
}
