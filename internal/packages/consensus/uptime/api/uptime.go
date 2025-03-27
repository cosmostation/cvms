package api

import (
	"context"
	"encoding/hex"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	commonparser "github.com/cosmostation/cvms/internal/common/parser"
	commontypes "github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/helper"
	sdkhelper "github.com/cosmostation/cvms/internal/helper/sdk"
	"github.com/cosmostation/cvms/internal/packages/consensus/uptime/types"
	"github.com/pkg/errors"
)

// TODO: Move parsing logic into parser module for other blockchains
// TODO: first parameter should change from indexer struct to interface
// TODO: Modify error wrapping
// query current staking validators
func getValidatorUptimeStatus(c common.CommonApp, chainName string, validators []commontypes.CosmosValidator, stakingValidators []commontypes.CosmosStakingValidator) (
	[]types.ValidatorUptimeStatus,
	error,
) {
	var (
		queryPathFunction func(string) string
		parser            func(resp []byte) (consensusAddress string, indexOffset float64, isTomstoned float64, missedBlocksCounter float64, err error)
	)

	switch chainName {
	// case "initia":
	// 	queryPath = types.InitiaStakingValidatorQueryPath(defaultStatus)
	// 	stakingValidatorParser = parser.InitiaStakingValidatorParser
	case "story":
		queryPathFunction = commontypes.StorySlashingQueryPath
		parser = commonparser.StorySlashingParser
	default:
		queryPathFunction = commontypes.CosmosSlashingQueryPath
		parser = commonparser.CosmosSlashingParser
	}

	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := c.APIClient.R().SetContext(ctx)

	// 2. extract bech32 valcons prefix using staking validator address
	var bech32ValconsPrefix string
	for idx, validator := range stakingValidators {
		exportedPrefix, err := sdkhelper.ExportBech32ValconsPrefix(validator.OperatorAddress)
		if err != nil {
			return nil, errors.Cause(err)
		}
		if idx == 0 {
			bech32ValconsPrefix = exportedPrefix
			break
		}
	}
	c.Debugf("bech32 valcons prefix: %s", bech32ValconsPrefix)

	// 3. make pubkey map by using consensus hex address with extracted valcons prefix
	pubkeysMap := make(map[string]string)
	vpMap := make(map[string]float64)
	for _, validator := range validators {
		bz, _ := hex.DecodeString(validator.Address)
		consensusAddress, err := sdkhelper.ConvertAndEncode(bech32ValconsPrefix, bz)
		if err != nil {
			return nil, common.ErrFailedConvertTypes
		}
		pubkeysMap[validator.Pubkey.Value] = consensusAddress

		vp, err := strconv.ParseFloat(validator.VotingPower, 64)
		if err != nil {
			return nil, common.ErrFailedConvertTypes
		}
		vpMap[validator.Pubkey.Value] = vp
	}

	// 4. Sort staking validators by vp
	orderedStakingValidators := sliceStakingValidatorByVP(stakingValidators, len(validators))

	// 5. init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	validatorResult := make([]types.ValidatorUptimeStatus, 0)
	wg.Add(len(orderedStakingValidators))

	for _, item := range orderedStakingValidators {
		// set query path
		moniker := item.Description.Moniker
		proposerAddress, _ := sdkhelper.ProposerAddressFromPublicKey(item.ConsensusPubkey.Key)
		validatorOperatorAddress := item.OperatorAddress
		consensusAddress := pubkeysMap[item.ConsensusPubkey.Key]
		queryPath := queryPathFunction(consensusAddress)
		vp := vpMap[item.ConsensusPubkey.Key]

		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(c.Entry)
			defer wg.Done()

			resp, err := requester.Get(queryPath)
			if err != nil {
				if resp == nil {
					ch <- helper.Result{Item: nil, Success: false}
					return
				}
				// c.Errorf("errors: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}
			if resp.StatusCode() != http.StatusOK {
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			_, _, isTomstoned, missedBlocksCounter, err := parser(resp.Body())
			if err != nil {
				c.Errorf("errors: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			ch <- helper.Result{
				Success: true,
				Item: types.ValidatorUptimeStatus{
					Moniker:                   moniker,
					ProposerAddress:           proposerAddress,
					ValidatorConsensusAddress: consensusAddress,
					MissedBlockCounter:        missedBlocksCounter,
					IsTomstoned:               isTomstoned,
					ValidatorOperatorAddress:  validatorOperatorAddress,
					VotingPower:               vp,
				}}
		}(ch)
		time.Sleep(10 * time.Millisecond)
	}

	// close channel
	go func() {
		wg.Wait()
		close(ch)
	}()

	// collect validator's orch
	errorCount := 0
	for r := range ch {
		if r.Success {
			validatorResult = append(validatorResult, r.Item.(types.ValidatorUptimeStatus))
			continue
		}
		errorCount++
	}

	if errorCount > 0 {
		c.Errorf("current errors count: %d", errorCount)
		return nil, common.ErrFailedHttpRequest
	}

	return validatorResult, nil
}

func getConsumerValidatorUptimeStatus(
	app common.CommonClient,
	providerValidators []commontypes.ProviderValidator,
	consumerValconsPrefix string,
) (
	[]types.ValidatorUptimeStatus,
	error,
) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := app.APIClient.R().SetContext(ctx)

	// 5. init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	validatorResult := make([]types.ValidatorUptimeStatus, 0)
	for _, pv := range providerValidators {
		wg.Add(1)
		// provider info
		moniker := pv.Description.Moniker
		providerValoperAddress := pv.ProviderValoperAddress
		providerValconsAddress := pv.PrvodierValconsAddress
		// consumer info
		proposerAddress, _ := sdkhelper.ProposerAddressFromPublicKey(pv.ConsumerKey.Pubkey)
		consumerValconsAddress, _ := sdkhelper.MakeValconsAddressFromPubeky(pv.ConsumerKey.Pubkey, consumerValconsPrefix)
		uptimeQueryPath := commontypes.CosmosSlashingQueryPath(consumerValconsAddress)

		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(app.Entry)
			defer wg.Done()

			resp, err := requester.Get(uptimeQueryPath)
			if err != nil {
				if resp == nil {
					ch <- helper.Result{Item: nil, Success: false}
					return
				}
				ch <- helper.Result{Item: nil, Success: false}
				return
			}
			if resp.StatusCode() != http.StatusOK {
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			_, _, isTomstoned, missedBlocksCounter, err := commonparser.CosmosSlashingParser(resp.Body())
			if err != nil {
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			ch <- helper.Result{
				Success: true,
				Item: types.ValidatorUptimeStatus{
					// provider
					Moniker:                   moniker,
					ValidatorConsensusAddress: providerValconsAddress,
					ValidatorOperatorAddress:  providerValoperAddress,
					IsTomstoned:               isTomstoned,
					// consumer
					ProposerAddress:          proposerAddress,
					ConsumerConsensusAddress: consumerValconsAddress,
					MissedBlockCounter:       missedBlocksCounter,
				}}
		}(ch)
		time.Sleep(10 * time.Millisecond)
	}

	// close channel
	go func() {
		wg.Wait()
		close(ch)
	}()

	// collect validator's orch
	errorCount := 0
	for r := range ch {
		if r.Success {
			validatorResult = append(validatorResult, r.Item.(types.ValidatorUptimeStatus))
			continue
		}
		errorCount++
	}

	if errorCount > 0 {
		app.Errorf("current errors count: %d", errorCount)
		return nil, common.ErrFailedHttpRequest
	}

	return validatorResult, nil
}

func getUptimeParams(c common.CommonClient, chainName string) (
	/* signed blocks window */ float64,
	/* minimum signed per window */ float64,
	/* unexpected error */ error,
) {
	var (
		queryPath string
		parser    func(resp []byte) (signedBlocksWindow float64, minSignedPerWindow float64, err error)
	)

	switch chainName {
	case "story":
		queryPath = commontypes.StorySlashingParamsQueryPath
		parser = commonparser.StorySlashingParamsParser
	default:
		queryPath = commontypes.CosmosSlashingParamsQueryPath
		parser = commonparser.CosmosSlashingParamsParser
	}

	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient.R().SetContext(ctx)
	resp, err := requester.Get(queryPath)
	if err != nil {
		return 0, 0, errors.Cause(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return 0, 0, errors.Errorf("api error: got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}

	signedBlocksWindow, minSignedPerWindow, err := parser(resp.Body())
	if err != nil {
		return 0, 0, errors.Cause(err)
	}

	c.Debugf("signed block window: %.f", signedBlocksWindow)
	c.Debugf("min signed per window: %.2f", minSignedPerWindow)
	return signedBlocksWindow, minSignedPerWindow, nil
}

func sliceStakingValidatorByVP(stakingValidators []commontypes.CosmosStakingValidator, totalConsensusValidators int) []commontypes.CosmosStakingValidator {
	sort.Slice(stakingValidators, func(i, j int) bool {
		tokensI, _ := strconv.ParseInt(stakingValidators[i].Tokens, 10, 64)
		tokensJ, _ := strconv.ParseInt(stakingValidators[j].Tokens, 10, 64)
		return tokensI > tokensJ // Sort in descending order
	})
	return stakingValidators[:totalConsensusValidators]
}
