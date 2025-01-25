package collector

import (
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/pkg/errors"

	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"
	"github.com/cosmostation/cvms/internal/packages/duty/axelar-evm/router"
	"github.com/cosmostation/cvms/internal/packages/duty/axelar-evm/types"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

// NOTE: this is for solo mode
var packageMonikers []string

const (
	Subsystem      = "axelar_evm"
	SubsystemSleep = 60 * time.Second
	UnHealthSleep  = 10 * time.Second

	MaintainerMetricName        = "maintainer_status"
	ActivatedEvmChainMetricName = "activated_chain"
)

func Start(p common.Packager) error {
	if ok := helper.Contains(types.SupportedChains, p.ChainName); ok {
		packageMonikers = p.Monikers
		for _, api := range p.APIs {
			exporter := common.NewExporter(p)
			exporter.SetAPIEndPoint(api)
			go loop(exporter, p)
			break
		}
		return nil
	}
	return errors.Errorf("unsupported chain type: %s", p.ProtocolType)
}

func loop(c *common.Exporter, p common.Packager) {
	rootLabels := common.BuildRootLabels(p)
	packageLabels := common.BuildPackageLabels(p)

	// each validators
	maintainerMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        MaintainerMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.ValidatorAddressLabel,
		common.MonikerLabel,
		common.EvmChainLabel,
	})

	// each chains
	activatedEVMChainMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		ConstLabels: packageLabels,
		Name:        ActivatedEvmChainMetricName,
	}, []string{common.EvmChainLabel})

	isUnhealth := false
	for {
		// node health check
		if isUnhealth {
			healthEndpoints := healthcheck.FilterHealthEndpoints(p.APIs, p.ProtocolType)
			for _, endpoint := range healthEndpoints {
				c.SetAPIEndPoint(endpoint)
				c.Infoln("client endpoint will be changed with health endpoint for this package")
				isUnhealth = false
				break
			}
			if len(healthEndpoints) == 0 {
				c.Errorln("failed to get any health endpoints from healthcheck filter, retry sleep 10s")
				time.Sleep(UnHealthSleep)
				continue
			}
		}

		status, err := router.GetStatus(c, p.ChainName)
		if err != nil {
			common.Health.With(rootLabels).Set(0)
			common.Ops.With(rootLabels).Inc()
			isUnhealth = true

			c.Logger.Errorf("failed to update metrics: %s", err.Error())
			time.Sleep(SubsystemSleep)

			continue
		}

		if p.Mode == common.NETWORK {
			// update metrics by each validators
			for _, item := range status.Validators {
				maintainerMetric.
					With(prometheus.Labels{
						common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
						common.EvmChainLabel:         item.EVMChainName,
						common.MonikerLabel:          item.Moniker,
					}).
					Set(float64(item.Status))
			}
		} else {
			// filter metrics for only specific validator
			for _, item := range status.Validators {
				if ok := helper.Contains(packageMonikers, item.Moniker); ok {
					maintainerMetric.
						With(prometheus.Labels{
							common.ValidatorAddressLabel: item.ValidatorOperatorAddress,
							common.EvmChainLabel:         item.EVMChainName,
							common.MonikerLabel:          item.Moniker,
						}).
						Set(item.Status)
				}
			}
		}

		// update metrics by each chain
		for _, evmChain := range status.ActiveEVMChains {
			activatedEVMChainMetric.
				With(prometheus.Labels{common.EvmChainLabel: evmChain}).
				Set(1)
		}

		c.Infof("updated %s metrics successfully and going to sleep %s ...", Subsystem, SubsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(SubsystemSleep)
	}
}
