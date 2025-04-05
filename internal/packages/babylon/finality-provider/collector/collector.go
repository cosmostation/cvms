package collector

import (
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"
	"github.com/cosmostation/cvms/internal/packages/babylon/finality-provider/api"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

const (
	Subsystem      = "babylon_finality_provider"
	SubsystemSleep = 10 * time.Second
	UnHealthSleep  = 10 * time.Second

	MissedVotesCounterMetricName = "missed_votes_counter"
	VotingPowerMetricName        = "voting_power"
	SignedVotesWindowMetricName  = "signed_votes_window"
	MinSignedPerWindowMetricName = "min_signed_per_window"

	LastFinalizedBlockMissingVotesCountMetricName = "last_finalized_block_missing_votes_count"
	LastFinalizedBlockMissingVPMetricName         = "last_finalized_block_missing_vp"
	LastFinalizedBlockFinalizedVPMetricName       = "last_finalized_block_finalized_vp"
	LastFinalizedBlockHeight                      = "last_finalized_block_height"

	METRIC_NAME_FINALITY_PROVIDERS_TOTAL = "total"
)

func Start(p common.Packager) error {
	if p.ChainName != "babylon" {
		return errors.New("unexpected chain for this package")
	}
	exporter := common.NewExporter(p)
	for _, rpc := range p.RPCs {
		exporter.SetRPCEndPoint(rpc)
		break
	}
	for _, api := range p.APIs {
		exporter.SetAPIEndPoint(api)
		break
	}
	exporter.Debugf("current mode: %v", p.Mode)
	go loop(exporter, p)
	return nil
}

func loop(exporter *common.Exporter, p common.Packager) {
	rootLabels := common.BuildRootLabels(p)
	packageLabels := common.BuildPackageLabels(p)

	// metrics for each validator
	uptimeMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        MissedVotesCounterMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.MonikerLabel,
		common.OrchestratorAddressLabel,
		common.BTCPKLabel,
		common.StatusLabel,
		common.ActiveLabel,
	})

	vpMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        VotingPowerMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.MonikerLabel,
		common.OrchestratorAddressLabel,
		common.BTCPKLabel,
		common.StatusLabel,
		common.ActiveLabel,
	})

	// metrics for each chain
	signedBlocksWindowMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        SignedVotesWindowMetricName,
		ConstLabels: packageLabels,
	})
	minSignedPerWindowMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        MinSignedPerWindowMetricName,
		ConstLabels: packageLabels,
	})

	lastFinalizedBlockMissingVotesCountMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        LastFinalizedBlockMissingVotesCountMetricName,
		ConstLabels: packageLabels,
	})
	lastFinalizedBlockMissingVPMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        LastFinalizedBlockMissingVPMetricName,
		ConstLabels: packageLabels,
	})
	lastFinalizedBlockFinalizedVPMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        LastFinalizedBlockFinalizedVPMetricName,
		ConstLabels: packageLabels,
	})
	lastFinalizedBlockHeightMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        LastFinalizedBlockHeight,
		ConstLabels: packageLabels,
	})

	fpTotalMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        METRIC_NAME_FINALITY_PROVIDERS_TOTAL,
		ConstLabels: packageLabels,
	}, []string{
		common.StatusLabel,
	})

	isUnhealth := false
	for {
		// node health check
		if isUnhealth {
			healthAPIs := healthcheck.FilterHealthEndpoints(p.APIs, p.ProtocolType)
			for _, api := range healthAPIs {
				exporter.SetAPIEndPoint(api)
				exporter.Infoln("client endpoint will be changed with health endpoint for this package")
				isUnhealth = false
				break
			}
			healthRPCs := healthcheck.FilterHealthRPCEndpoints(p.RPCs, p.ProtocolType)
			for _, rpc := range healthRPCs {
				exporter.SetRPCEndPoint(rpc)
				exporter.Warnf("RPC endpoint will be changed with health endpoint for this package: %s", rpc)
				isUnhealth = false
				break
			}
			if len(healthAPIs) == 0 || len(healthRPCs) == 0 {
				isUnhealth = true
				exporter.Errorln("failed to get any health endpoints from healthcheck filter, retry sleep 10s")
				time.Sleep(UnHealthSleep)
				continue
			}
		}

		status, err := api.GetFinalityProviderUptime(exporter)
		if err != nil {
			common.Health.With(rootLabels).Set(0)
			common.Ops.With(rootLabels).Inc()
			isUnhealth = true

			exporter.Errorf("failed to update metrics err: %s and going to sleep %s...", err, SubsystemSleep.String())
			time.Sleep(SubsystemSleep)

			continue
		}

		exporter.Debugf("got total %d status", len(status.FinalityProvidersStatus))

		// Reset metrics to avoid stale label combinations (e.g., jailed status change)
		uptimeMetric.Reset()
		vpMetric.Reset()

		if p.Mode == common.NETWORK {
			// update metrics by each validators
			for _, item := range status.FinalityProvidersStatus {
				uptimeMetric.
					With(prometheus.Labels{
						common.MonikerLabel:             item.Moniker,
						common.BTCPKLabel:               item.BTCPK,
						common.OrchestratorAddressLabel: item.Address,
						common.StatusLabel:              item.Status,
						common.ActiveLabel:              item.Active,
					}).
					Set(float64(item.MissedBlockCounter))

				if item.VotingPower != 0 {
					vpMetric.
						With(prometheus.Labels{
							common.MonikerLabel:             item.Moniker,
							common.BTCPKLabel:               item.BTCPK,
							common.OrchestratorAddressLabel: item.Address,
							common.StatusLabel:              item.Status,
							common.ActiveLabel:              item.Active,
						}).
						Set(item.VotingPower)
				}
			}

			fpTotalMetric.With(prometheus.Labels{common.StatusLabel: "active"}).Set(float64(status.FinalityProviderTotal.Active))
			fpTotalMetric.With(prometheus.Labels{common.StatusLabel: "inactive"}).Set(float64(status.FinalityProviderTotal.Inactive))
			fpTotalMetric.With(prometheus.Labels{common.StatusLabel: "jailed"}).Set(float64(status.FinalityProviderTotal.Jailed))
			fpTotalMetric.With(prometheus.Labels{common.StatusLabel: "slashed"}).Set(float64(status.FinalityProviderTotal.Slashed))
		} else {
			for _, item := range status.FinalityProvidersStatus {
				if ok := helper.Contains(exporter.Monikers, item.Moniker); ok {
					uptimeMetric.
						With(prometheus.Labels{
							common.MonikerLabel:             item.Moniker,
							common.BTCPKLabel:               item.BTCPK,
							common.OrchestratorAddressLabel: item.Address,
							common.StatusLabel:              item.Status,
							common.ActiveLabel:              item.Active,
						}).
						Set(float64(item.MissedBlockCounter))

					if item.VotingPower != 0 {
						vpMetric.
							With(prometheus.Labels{
								common.MonikerLabel:             item.Moniker,
								common.BTCPKLabel:               item.BTCPK,
								common.OrchestratorAddressLabel: item.Address,
								common.StatusLabel:              item.Status,
								common.ActiveLabel:              item.Active,
							}).
							Set(item.VotingPower)
					}
				}
			}
		}

		// update metrics by each chain
		signedBlocksWindowMetric.Set(status.SignedBlocksWindow)
		minSignedPerWindowMetric.Set(status.MinSignedPerWindow)

		// Add a new logic for providing last finalized block status
		lastFinalizedBlockMissingVotesCountMetric.Set(status.LastFinalizedBlockInfo.MissingVotes)
		lastFinalizedBlockMissingVPMetric.Set(status.LastFinalizedBlockInfo.MissingVP)
		lastFinalizedBlockFinalizedVPMetric.Set(status.LastFinalizedBlockInfo.FinalizedVP)
		lastFinalizedBlockHeightMetric.Set(status.LastFinalizedBlockInfo.BlockHeight)

		exporter.Infof("updated metrics successfully and going to sleep %s ...", SubsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(SubsystemSleep)
	}
}
