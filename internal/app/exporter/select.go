package exporter

import (
	"github.com/pkg/errors"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/helper/config"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	// babylon
	btcdelegations "github.com/cosmostation/cvms/internal/packages/babylon/btc-delegations"
	fpcollector "github.com/cosmostation/cvms/internal/packages/babylon/finality-provider/collector"

	// validator consensus packages
	uptime "github.com/cosmostation/cvms/internal/packages/consensus/uptime/collector"

	// validator duty packages
	axelarevm "github.com/cosmostation/cvms/internal/packages/duty/axelar-evm/collector"
	eventnonce "github.com/cosmostation/cvms/internal/packages/duty/eventnonce/collector"
	oracle "github.com/cosmostation/cvms/internal/packages/duty/oracle/collector"
	yoda "github.com/cosmostation/cvms/internal/packages/duty/yoda/collector"

	// health packages
	block "github.com/cosmostation/cvms/internal/packages/health/block/collector"

	// utility packages
	balance "github.com/cosmostation/cvms/internal/packages/utility/balance/collector"
	upgrade "github.com/cosmostation/cvms/internal/packages/utility/upgrade/collector"
	// TODO: in the future, we need to implement EVM contract & WASM contract statement for validators
	// contract "github.com/cosmostation/cvms/internal/packages/contract/collector"
)

func selectPackage(
	m common.Mode, f promauto.Factory, l *logrus.Logger,
	mainnet bool, chainID, chainName, pkg, protocolType string,
	balanceDenom string, balanceExponent int, isConsumer bool,
	cc config.ChainConfig, monikers []string,
) error {
	// Add validation logic on each provided URL
	validAPIs := make([]string, 0)
	validRPCs := make([]string, 0)
	validGRPCs := make([]string, 0)

	for _, node := range cc.Nodes {
		if helper.ValidateURL(node.RPC) {
			validRPCs = append(validRPCs, node.RPC)
		}

		if helper.ValidateURL(node.API) {
			validAPIs = append(validAPIs, node.API)
		}

		// not found how to check validation for GRPC endpoint
		validGRPCs = append(validGRPCs, node.GRPC)
	}

	providerRPCs := make([]string, 0)
	providerAPIs := make([]string, 0)
	if isConsumer {
		for _, node := range cc.ProviderNodes {
			if helper.ValidateURL(node.RPC) {
				providerRPCs = append(providerRPCs, node.RPC)
			}
			if helper.ValidateURL(node.API) {
				providerAPIs = append(providerAPIs, node.API)
			}
		}
	}

	switch {
	case pkg == "block":
		endpoints := common.Endpoints{RPCs: validRPCs, CheckRPC: true}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return block.Start(*p)
	case pkg == "upgrade":
		endpoints := common.Endpoints{RPCs: validRPCs, CheckRPC: true, APIs: validAPIs, CheckAPI: true}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return upgrade.Start(*p)
	case pkg == "balance":
		endpoints := common.Endpoints{APIs: validAPIs, CheckAPI: true}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		p.SetInfoForBalancePackage(cc.TrackingAddresses, balanceDenom, balanceExponent)
		return balance.Start(*p)
	case pkg == "oracle":
		endpoints := common.Endpoints{APIs: validAPIs, CheckAPI: true}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints, monikers...)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return oracle.Start(*p)
	case pkg == "eventnonce":
		endpoints := common.Endpoints{GRPCs: validGRPCs, CheckGRPC: true}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints, monikers...)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return eventnonce.Start(*p)
	case pkg == "yoda":
		endpoints := common.Endpoints{RPCs: validRPCs, CheckRPC: true, APIs: validAPIs, CheckAPI: true}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints, monikers...)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return yoda.Start(*p)
	case pkg == "axelar-evm":
		endpoints := common.Endpoints{APIs: validAPIs, CheckAPI: true}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints, monikers...)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return axelarevm.Start(*p)
	case pkg == "uptime":
		endpoints := common.Endpoints{
			RPCs: validRPCs, CheckRPC: true,
			APIs: validAPIs, CheckAPI: true,
		}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints, monikers...)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		if isConsumer {
			providerEndpoints := common.Endpoints{RPCs: providerRPCs, CheckRPC: true, APIs: providerAPIs, CheckAPI: true}
			p.SetAddtionalEndpoints(providerEndpoints)
			p.SetConsumer()
		}
		return uptime.Start(*p)
	case pkg == "babylon-finality-provider-uptime":
		endpoints := common.Endpoints{
			RPCs: validRPCs, CheckRPC: true,
			APIs: validAPIs, CheckAPI: true,
		}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints, monikers...)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return fpcollector.Start(*p)
	case pkg == "babylon-btc-delegations":
		endpoints := common.Endpoints{
			RPCs: validRPCs, CheckRPC: true,
			APIs: validAPIs, CheckAPI: true,
		}
		p, err := common.NewPackager(m, f, l, mainnet, chainID, chainName, pkg, protocolType, cc, endpoints, monikers...)
		if err != nil {
			return errors.Wrap(err, common.ErrFailedToBuildPackager)
		}
		return btcdelegations.Start(*p)
	}
	// NOTE: contract package is not using now, but it could be enabled if it needs
	// case strings.Contains(packageName, "contract"):
	// 	return common.ErrUnSupportedPackage

	// NOTE: if you want to add internal service exporter, please use this sector.
	// case packageName == "something-internal-service":
	// 1. create a library for your internal service, by using references, our packages
	// 2. import.. and add the library package here
	// 3. return internal.Start(*metric)
	// return common.ErrUnSupportedPackage

	return common.ErrUnSupportedPackage
}
