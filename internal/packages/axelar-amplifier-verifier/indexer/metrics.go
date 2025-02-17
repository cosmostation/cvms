package indexer

import (
	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	PollMetricName = "poll"
	PollLabel      = "poll_id"

	PollVoteMetricName = "poll_vote"
	VoteStatusLabel    = "status"
	VerifierLabel      = "verifier"
)

func (idx *AxelarAmplifierVerifierIndexer) initLabelsAndMetrics() {
	idx.MetricsMap[common.IndexPointerBlockHeightMetricName] = idx.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   subsystem,
		Name:        common.IndexPointerBlockHeightMetricName,
		ConstLabels: idx.PackageLabels,
	})

	idx.MetricsMap[common.IndexPointerBlockTimestampMetricName] = idx.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   subsystem,
		Name:        common.IndexPointerBlockTimestampMetricName,
		ConstLabels: idx.PackageLabels,
	})

	latestBlockHeightMetric := idx.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   subsystem,
		Name:        common.LatestBlockHeightMetricName,
		ConstLabels: idx.PackageLabels,
	})
	latestBlockHeightMetric.Set(0)
	idx.MetricsMap[common.LatestBlockHeightMetricName] = latestBlockHeightMetric

	// only axelar amplifier verifier
	idx.MetricsCountVecMap[PollMetricName] = idx.Factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   common.Namespace,
			Subsystem:   subsystem,
			Name:        PollMetricName,
			ConstLabels: idx.PackageLabels},
		[]string{
			PollLabel,
		})

	idx.MetricsCountVecMap[PollVoteMetricName] = idx.Factory.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   common.Namespace,
			Subsystem:   subsystem,
			Name:        PollVoteMetricName,
			ConstLabels: idx.PackageLabels},
		[]string{
			PollLabel,
			VoteStatusLabel,
			VerifierLabel,
		})
}

func (idx *AxelarAmplifierVerifierIndexer) updatePrometheusMetrics(indexPointer int64, pollMap PollMap) {
	for poll, votes := range pollMap {
		idx.MetricsCountVecMap[PollMetricName].With(prometheus.Labels{PollLabel: poll}).Inc()
		for _, v := range votes {
			_, exist := idx.VAM[v.VerifierID]
			if !exist {
				idx.Panicln(idx.VAM, v.VerifierID)
			}
			idx.MetricsCountVecMap[PollVoteMetricName].
				With(prometheus.Labels{
					PollLabel:       poll,
					VoteStatusLabel: v.Status.ToString(),
					VerifierLabel:   idx.VAM[v.VerifierID],
				}).
				Inc()
		}
	}
	idx.MetricsMap[common.IndexPointerBlockHeightMetricName].Set(float64(indexPointer))
	_, timestamp, _, _, _, _, err := api.GetBlock(idx.CommonClient, indexPointer)
	if err != nil {
		idx.Errorf("failed to get block %d: %s", indexPointer, err)
		return
	}
	idx.MetricsMap[common.IndexPointerBlockTimestampMetricName].Set((float64(timestamp.Unix())))
	idx.Debugf("update prometheus metrics %d height", indexPointer)
}
