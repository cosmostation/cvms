package collector

import (
	"time"

	"github.com/pkg/errors"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/duty/eventnonce/router"
	"github.com/cosmostation/cvms/internal/packages/duty/eventnonce/types"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

// NOTE: this is for solo mode
var packageMonikers []string

const (
	Subsystem      = "eventnonce"
	SubsystemSleep = 60 * time.Second
	UnHealthSleep  = 10 * time.Second

	NonceMetricName         = "nonce"
	HeighestNonceMetricName = "highest_nonce"
)

func Start(p common.Packager) error {
	if ok := helper.Contains(types.SupportedChains, p.ChainName); ok {
		packageMonikers = p.Monikers
		for _, baseURL := range p.GRPCs {
			client := common.NewExporter(p)
			client.SetGRPCEndPoint(baseURL)
			go loop(client, p)
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
	eventNonceMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        NonceMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.ValidatorAddressLabel,
		common.MonikerLabel,
		common.OrchestratorAddressLabel,
	})

	// each chain
	heighestNonceMetric := p.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        HeighestNonceMetricName,
		ConstLabels: packageLabels,
	})

	isUnhealth := false
	for {
		// node health check
		if isUnhealth {
			// TODO: Add healthcheck logic for GRPC endpoints
			// healthEndpoints := healthcheck.FilterHealthEndpoints(p.GRPCs, p.ProtocolType)
			for _, grpc := range p.GRPCs {
				if grpc != c.GetGRPCEndPoint() {
					c.SetGRPCEndPoint(grpc)
					c.Infoln("client endpoint will be changed with health endpoint for this package")
					isUnhealth = false
					break
				}
			}
			// if len(healthEndpoints) == 0 {
			// 	c.Errorln("failed to get any health endpoints from healthcheck filter, retry sleep 10s")
			// 	time.Sleep(UnHealthSleep)
			// 	// health check로 넣어야 할 듯.
			// 	continue
			// }
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
				eventNonceMetric.
					With(prometheus.Labels{
						common.ValidatorAddressLabel:    item.ValidatorOperatorAddress,
						common.OrchestratorAddressLabel: item.OrchestratorAddress,
						common.MonikerLabel:             item.Moniker,
					}).
					Set(float64(item.EventNonce))
			}
		} else {
			// filter metrics for only specific validator
			for _, item := range status.Validators {
				if ok := helper.Contains(packageMonikers, item.Moniker); ok {
					eventNonceMetric.
						With(prometheus.Labels{
							common.ValidatorAddressLabel:    item.ValidatorOperatorAddress,
							common.OrchestratorAddressLabel: item.OrchestratorAddress,
							common.MonikerLabel:             item.Moniker,
						}).
						Set(float64(item.EventNonce))
				}
			}
		}
		// update metrics by each chain
		heighestNonceMetric.Set(status.HeighestNonce)

		c.Infof("updated %s metrics successfully and going to sleep %s ...", Subsystem, SubsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(SubsystemSleep)
	}
}
