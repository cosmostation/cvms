package collector

import (
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"
	"github.com/cosmostation/cvms/internal/packages/utility/balance/router"
	"github.com/cosmostation/cvms/internal/packages/utility/balance/types"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

const (
	Subsystem      = "balance"
	SubsystemSleep = 60 * time.Second
	UnHealthSleep  = 10 * time.Second

	AmountMetricName = "remaining_amount"
)

func Start(p common.Packager) error {
	if ok := helper.Contains(types.SupportedProtocolTypes, p.ProtocolType); ok {
		for _, baseURL := range p.APIs {
			client := common.NewExporter(p)
			client.SetAPIEndPoint(baseURL)
			go loop(client, p)
			return nil
		}
	}
	return errors.Errorf("unsupported chain type: %s", p.ProtocolType)
}

func loop(c *common.Exporter, p common.Packager) {
	rootLabels := common.BuildRootLabels(p)
	packageLabels := common.BuildPackageLabels(p)

	amountMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        AmountMetricName,
		ConstLabels: packageLabels},
		[]string{
			common.BalanceAddressLabel,
		},
	)

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

		// collect status
		status, err := router.GetStatus(c, p)
		if err != nil {
			common.Health.With(rootLabels).Set(0)
			common.Ops.With(rootLabels).Inc()
			isUnhealth = true

			c.Logger.Errorf("failed to update metrics: %s", err.Error())
			time.Sleep(SubsystemSleep)

			continue
		}

		for _, item := range status.Balances {
			amountMetric.
				With(prometheus.Labels{common.BalanceAddressLabel: item.Address}).
				Set(item.RemainingBalance)
		}

		c.Infof("updated %s metrics successfully and going to sleep %s ...", SubsystemSleep, SubsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(SubsystemSleep)
	}
}
