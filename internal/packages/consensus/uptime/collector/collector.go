package collector

import (
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"
	"github.com/cosmostation/cvms/internal/packages/consensus/uptime/router"
	"github.com/cosmostation/cvms/internal/packages/consensus/uptime/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

const (
	Subsystem                    = "uptime"
	SubsystemSleep               = 10 * time.Second
	UnHealthSleep                = 10 * time.Second
	MissBlockCounterMetricName   = "missed_blocks_counter"
	JailedMetricName             = "jailed"
	SignedBlocksWindowMetricName = "signed_blocks_window"
	MinSignedPerWindowMetricName = "min_signed_per_window"

	METRIC_NAME_VP = "validator_voting_power"
)

func Start(p common.Packager) error {
	if ok := helper.Contains(types.SupportedProtocolTypes, p.ProtocolType); ok {
		exporter := common.NewExporter(p)
		for _, rpc := range p.RPCs {
			exporter.SetRPCEndPoint(rpc)
			break
		}
		for _, api := range p.APIs {
			exporter.SetAPIEndPoint(api)
			break
		}

		if p.IsConsumerChain {
			exporter.OptionalClient = common.NewOptionalClient(exporter.Entry)
			for _, rpc := range p.ProviderEndPoints.RPCs {
				exporter.OptionalClient.SetRPCEndPoint(rpc)
				break
			}
			for _, api := range p.ProviderEndPoints.APIs {
				exporter.OptionalClient.SetAPIEndPoint(api)
				break
			}
		}
		go loop(exporter, p)
		return nil
	}
	return errors.Errorf("unsupported protocol type: %s", p.ProtocolType)
}

func loop(exporter *common.Exporter, p common.Packager) {
	rootLabels := common.BuildRootLabels(p)
	packageLabels := common.BuildPackageLabels(p)

	// metrics for each validator
	uptimeMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        MissBlockCounterMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.MonikerLabel,
		common.ValidatorAddressLabel,
		common.ConsensusAddressLabel,
		common.ProposerAddressLabel,
	})

	jailedMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        JailedMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.MonikerLabel,
		common.ValidatorAddressLabel,
		common.ConsensusAddressLabel,
		common.ProposerAddressLabel,
	})

	vpMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        METRIC_NAME_VP,
		ConstLabels: packageLabels,
	}, []string{
		common.MonikerLabel,
		common.ValidatorAddressLabel,
		common.ConsensusAddressLabel,
		common.ProposerAddressLabel,
	})

	// metrics for each chain
	signedBlocksWindowMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        SignedBlocksWindowMetricName,
		ConstLabels: packageLabels,
	})
	minSignedPerWindowMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        MinSignedPerWindowMetricName,
		ConstLabels: packageLabels,
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

		status, err := router.GetStatus(exporter, p)
		if err != nil {
			common.Health.With(rootLabels).Set(0)
			common.Ops.With(rootLabels).Inc()
			isUnhealth = true

			exporter.Errorf("failed to update metrics err: %s and going to sleep %s...", err, SubsystemSleep.String())
			time.Sleep(SubsystemSleep)

			continue
		}

		if p.Mode == common.NETWORK {
			// update metrics by each validators
			for _, item := range status.Validators {
				uptimeMetric.
					With(prometheus.Labels{
						common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
						common.ConsensusAddressLabel: item.ValidatorConsensusAddress,
						common.ProposerAddressLabel:  item.ProposerAddress,
						common.MonikerLabel:          item.Moniker,
					}).
					Set(float64(item.MissedBlockCounter))
				jailedMetric.
					With(prometheus.Labels{
						common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
						common.ConsensusAddressLabel: item.ValidatorConsensusAddress,
						common.ProposerAddressLabel:  item.ProposerAddress,
						common.MonikerLabel:          item.Moniker,
					}).
					Set(float64(item.IsTomstoned))
				vpMetric.
					With(prometheus.Labels{
						common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
						common.ConsensusAddressLabel: item.ValidatorConsensusAddress,
						common.ProposerAddressLabel:  item.ProposerAddress,
						common.MonikerLabel:          item.Moniker,
					}).
					Set(item.VotingPower)
			}
		} else {
			// update metrics by each validators
			for _, item := range status.Validators {
				if ok := helper.Contains(p.Monikers, item.Moniker); ok {
					uptimeMetric.
						With(prometheus.Labels{
							common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
							common.ConsensusAddressLabel: item.ValidatorConsensusAddress,
							common.ProposerAddressLabel:  item.ProposerAddress,
							common.MonikerLabel:          item.Moniker,
						}).
						Set(float64(item.MissedBlockCounter))
					jailedMetric.
						With(prometheus.Labels{
							common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
							common.ConsensusAddressLabel: item.ValidatorConsensusAddress,
							common.ProposerAddressLabel:  item.ProposerAddress,
							common.MonikerLabel:          item.Moniker,
						}).
						Set(float64(item.IsTomstoned))
					vpMetric.
						With(prometheus.Labels{
							common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
							common.ConsensusAddressLabel: item.ValidatorConsensusAddress,
							common.ProposerAddressLabel:  item.ProposerAddress,
							common.MonikerLabel:          item.Moniker,
						}).
						Set(item.VotingPower)
				}
			}
		}

		// update metrics by each chain
		signedBlocksWindowMetric.Set(status.SignedBlocksWindow)
		minSignedPerWindowMetric.Set(status.MinSignedPerWindow)

		exporter.Infof("updated metrics successfully and going to sleep %s ...", SubsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(SubsystemSleep)
	}
}
