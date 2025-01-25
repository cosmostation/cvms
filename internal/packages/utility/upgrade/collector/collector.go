package collector

import (
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"
	"github.com/cosmostation/cvms/internal/packages/utility/upgrade/router"
	"github.com/cosmostation/cvms/internal/packages/utility/upgrade/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

const (
	Subsystem      = "upgrade"
	subsystemSleep = 60 * time.Second
	UnHealthSleep  = 10 * time.Second

	RemainingTimeMetricName   = "remaining_time"
	RemainingBlocksMetricName = "remaining_blocks"
)

func Start(p common.Packager) error {
	if ok := helper.Contains(types.SupportedProtocolTypes, p.ProtocolType); ok {
		exporter := common.NewExporter(p)
		// setup endpoints
		for _, rpc := range p.RPCs {
			exporter.SetRPCEndPoint(rpc)
			break
		}
		for _, api := range p.APIs {
			exporter.SetAPIEndPoint(api)
			break
		}
		go loop(exporter, p)
		return nil
	}
	return errors.Errorf("unsupported chain type: %s", p.ProtocolType)
}

func loop(c *common.Exporter, p common.Packager) {
	rootLabels := common.BuildRootLabels(p)
	packageLabels := common.BuildPackageLabels(p)

	// TODO: active node 개수? -> health package 구현
	remainingUpgradeTimeMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        RemainingTimeMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.UpgradeNameLabel,
	})
	RemainingBlocksMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        RemainingBlocksMetricName,
		ConstLabels: packageLabels,
	}, []string{
		common.UpgradeNameLabel,
	})

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
			if err == common.ErrCanSkip {
				common.Health.With(rootLabels).Set(1)
				common.Ops.With(rootLabels).Inc()

				c.Infof("updated %s metrics successfully and going to sleep %s ...", Subsystem, subsystemSleep.String())
				c.Infoln("there is no any upgrade in the onchain ,so reset upgrade metrics")
				remainingUpgradeTimeMetric.Reset()

				time.Sleep(subsystemSleep)
				continue
			}
			common.Health.With(rootLabels).Set(0)
			common.Ops.With(rootLabels).Inc()
			isUnhealth = true

			c.Errorf("failed to update metrics: %s", err.Error())
			time.Sleep(subsystemSleep)
			continue
		}

		// NOTE: upgrade package ...
		remainingUpgradeTimeMetric.
			With(prometheus.Labels{common.UpgradeNameLabel: status.UpgradeName}).
			Set(status.RemainingTime)
		RemainingBlocksMetric.
			With(prometheus.Labels{common.UpgradeNameLabel: status.UpgradeName}).
			Set(status.RemainingBlocks)

		c.Infof("updated %s metrics successfully and going to sleep %s ...", Subsystem, subsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(subsystemSleep)
	}
}
