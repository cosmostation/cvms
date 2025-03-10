package btcdelegation

import (
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper/healthcheck"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ common.CollectorStart = Start
	_ common.CollectorLoop  = loop
)

const (
	Subsystem      = "babylon_btc_delegations"
	SubsystemSleep = 10 * time.Second
	UnHealthSleep  = 10 * time.Second

	TotalDelegationsMetricName = "total"
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

	totalMetric := p.Factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   common.Namespace,
		Subsystem:   Subsystem,
		Name:        TotalDelegationsMetricName,
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

		status, err := GetBabylonBTCDelegationsStatus(exporter)
		if err != nil {
			common.Health.With(rootLabels).Set(0)
			common.Ops.With(rootLabels).Inc()
			isUnhealth = true

			exporter.Errorf("failed to update metrics err: %s and going to sleep %s...", err, SubsystemSleep.String())
			time.Sleep(SubsystemSleep)

			continue
		}

		for key, value := range status {
			totalMetric.With(prometheus.Labels{common.StatusLabel: key}).Set(float64(value))
		}

		exporter.Infof("updated metrics successfully and going to sleep %s ...", SubsystemSleep.String())

		// update health and ops
		common.Health.With(rootLabels).Set(1)
		common.Ops.With(rootLabels).Inc()

		// sleep
		time.Sleep(SubsystemSleep)
	}
}
