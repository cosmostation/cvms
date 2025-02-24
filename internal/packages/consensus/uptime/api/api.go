package api

import (
	"github.com/cosmostation/cvms/internal/common"
	commonapi "github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/packages/consensus/uptime/types"
	"github.com/pkg/errors"
)

func GetUptimeStatus(exporter *common.Exporter) (types.CommonUptimeStatus, error) {
	// 1. get staking validators
	stakingValidators, err := commonapi.GetStakingValidators(exporter.CommonClient, exporter.ChainName)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}
	exporter.Debugf("got total staking validators: %d", len(stakingValidators))

	// 2. get (consensus) validators
	validators, err := commonapi.GetValidators(exporter.CommonClient)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}
	exporter.Debugf("got total consensus validators: %d", len(validators))

	// 3. get validators' uptime status
	validatorUptimeStatus, err := getValidatorUptimeStatus(exporter.CommonApp, exporter.ChainName, validators, stakingValidators)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}
	exporter.Debugf("got total validator uptime: %d", len(validatorUptimeStatus))

	// 4. get on-chain uptime parameter
	signedBlocksWindow, minSignedPerWindow, err := getUptimeParams(exporter.CommonClient, exporter.ChainName)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}

	return types.CommonUptimeStatus{
		SignedBlocksWindow: signedBlocksWindow,
		MinSignedPerWindow: minSignedPerWindow,
		Validators:         validatorUptimeStatus,
	}, nil
}

func GetConsumserUptimeStatus(exporter *common.Exporter, chainID string) (types.CommonUptimeStatus, error) {
	// set provider client
	providerClient := exporter.OptionalClient
	consumerClient := exporter.CommonClient

	// 1. get consumer id by using chain-id
	var consumerID string
	consumerChains, err := commonapi.GetConsumerChainID(providerClient)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Wrap(err, "failed to get consumer chain id")
	}
	for _, consumerChain := range consumerChains {
		if consumerChain.ChainID == chainID {
			consumerID = consumerChain.ConsumerID
			break
		}
	}
	// validation check
	if consumerID == "" {
		return types.CommonUptimeStatus{}, errors.Errorf("failed to find consumer id, check again your chain-id: %s", chainID)
	}
	exporter.Debugf("got consumer id: %s", consumerID)

	// 2. get provider validators
	providerStakingValidators, err := commonapi.GetProviderValidators(providerClient, consumerID)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}
	exporter.Debugf("got total provider staking validators: %d", len(providerStakingValidators))

	// 3. get hrp via slashing info
	hrp, err := commonapi.GetConsumerChainHRP(consumerClient)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}
	exporter.Debugf("got hrp for making valcons address: %s", hrp)

	// 4. get consumer validators uptime status
	validatorUptimeStatus, err := getConsumerValidatorUptimeStatus(consumerClient, providerStakingValidators, hrp)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}
	exporter.Debugf("got total consumer validator uptime: %d", len(validatorUptimeStatus))
	// 5. get on-chain slashing parameter
	signedBlocksWindow, minSignedPerWindow, err := getUptimeParams(consumerClient, exporter.ChainName)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Cause(err)
	}

	// 6. get consumer channel uptime status
	consumerUptimeStatus := &types.ConsumerUptimeStatus{}
	channelStatus, err := getLastCCVUpdate(consumerClient)
	if err != nil {
		return types.CommonUptimeStatus{}, errors.Wrap(err, "failed to get CCV channel status")
	}
	consumerUptimeStatus.LastCCVUpdate = channelStatus

	return types.CommonUptimeStatus{
		SignedBlocksWindow:   signedBlocksWindow,
		MinSignedPerWindow:   minSignedPerWindow,
		Validators:           validatorUptimeStatus,
		ConsumerUptimeStatus: consumerUptimeStatus,
	}, nil
}
