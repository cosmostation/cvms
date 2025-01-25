package collector

import (
	"time"

	"github.com/pkg/errors"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/health/block/router"
	"github.com/cosmostation/cvms/internal/packages/health/block/types"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

const (
	Subsystem      = "block"
	subsystemSleep = 15 * time.Second

	TimestampMetricName   = "timestamp"
	BlockHeightMetricName = "height"
)

func Start(p common.Packager) error {
	if ok := helper.Contains(types.SupportedChainTypes, p.ProtocolType); ok {
		for _, rpc := range p.RPCs {
			client := common.NewExporter(p)
			client.SetRPCEndPoint(rpc)
			go loop(client, p)
		}
		return nil
	}
	return errors.Errorf("unsupported chain type: %s", p.ProtocolType)
}

func loop(c *common.Exporter, m common.Packager) {
	rootLabels := common.BuildRootLabels(m)
	packageLabels := common.BuildPackageLabelsWithURL(m, c.GetRPCEndPoint())

	timestampMetric := m.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		ConstLabels: packageLabels,
		Name:        TimestampMetricName,
	})
	blockHeightMetric := m.Factory.NewGauge(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		ConstLabels: packageLabels,
		Name:        BlockHeightMetricName,
	})

	for {
		// NOTE: block is a default package, so skip the select node logic to GetStatus method
		// check node status and change node if it needed...
		status, err := router.GetStatus(c, m.ProtocolType)
		if err != nil {
			// skip updating metrics
			c.Errorf("failed to update metrics err: %s", err.Error())

			common.Health.With(rootLabels).Set(0)
			common.Ops.With(rootLabels).Inc()

			time.Sleep(subsystemSleep)
			continue
		}

		// NOTE: block package is a default package, so the metrics will be updated regardless of app mode
		timestampMetric.Set(status.LastBlockTimeStamp)
		blockHeightMetric.Set(status.LastBlockHeight)

		c.Infof("updated metrics successfully and going to sleep %s ...", subsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(subsystemSleep)
	}
}
