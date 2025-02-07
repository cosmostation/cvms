package api

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/parser"
	"github.com/cosmostation/cvms/internal/common/types"
	sdkhelper "github.com/cosmostation/cvms/internal/helper/sdk"
	"github.com/pkg/errors"
)

func GetStatus(c common.CommonClient) (
	/* block height */ int64,
	/* block timestamp */ time.Time,
	error,
) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// requester := c.RPCClient.R().SetContext(ctx)
	resp, err := c.RPCClient.Get(ctx, types.CosmosStatusQueryPath)
	if err != nil {
		return 0, time.Time{}, errors.Errorf("rpc call is failed from: %s", err)
	}
	// if resp.StatusCode() != http.StatusOK {
	// 	return 0, time.Time{}, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
	// }

	latestBlockHeight, latestBlockTimestamp, err := parser.CosmosStatusParser(resp)
	if err != nil {
		return 0, time.Time{}, errors.Wrapf(err, "got data, but failed to parse the data")
	}

	return latestBlockHeight, latestBlockTimestamp, nil
}

// query a new block to find missed validators index
func GetBlock(c common.CommonClient, height int64) (
	/* block height */ int64,
	/* block timestamp */ time.Time,
	/* block proposer addrss */ string,
	/* block txs */ []types.Tx,
	/* last commit block height*/ int64,
	/* block validators signatures */ []types.Signature,
	error,
) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	// requester := c.RPCClient.R().SetContext(ctx)

	resp, err := c.RPCClient.Get(ctx, types.CosmosBlockQueryPath(height))
	if err != nil {
		return 0, time.Time{}, "", nil, 0, nil, errors.Errorf("rpc call is failed from: %s", err)
	}
	// if resp.StatusCode() != http.StatusOK {
	// 	return 0, time.Time{}, "", nil, 0, nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
	// }

	blockHeight, blockTimeStamp, blockProposerAddress, blockTxs, lastCommitBlockHeight, blockSignatures, err := parser.CosmosBlockParser(resp)
	if err != nil {
		return 0, time.Time{}, "", nil, 0, nil, errors.Wrapf(err, "got data, but failed to parse the data")
	}

	return blockHeight, blockTimeStamp, blockProposerAddress, blockTxs, lastCommitBlockHeight, blockSignatures, nil
}

// query cosmos validators on each a new block
func GetValidators(c common.CommonClient, height ...int64) ([]types.CosmosValidator, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	totalValidators := make([]types.CosmosValidator, 0)
	var queryPath string
	page := 1
	pageAddress := &page
	maxPage := 5

	for {
		if len(height) > 0 {
			queryPath = types.CosmosValidatorQueryPathWithHeight(height[0], page)
		} else {
			queryPath = types.CosmosValidatorQueryPath(page)
		}

		resp, err := c.RPCClient.Get(ctx, queryPath)
		if err != nil {
			return nil, errors.Errorf("rpc call is failed from: %s", err)
		}
		// if resp.StatusCode() != http.StatusOK {
		// 	return nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
		// }

		validators, totalValidatorsCount, err := parser.CosmosValidatorParser(resp)
		if err != nil {
			return nil, errors.Wrapf(err, "got data, but failed to parse the data")
		}

		c.Debugf("found cosmos validators: %d in %d page", len(validators), page)
		totalValidators = append(totalValidators, validators...)

		// if it was already found total validators in the for loop, break the loop and return
		if len(totalValidators) == int(totalValidatorsCount) {
			c.Debugf("found all cosmos validators %d, who matched each staking validator", len(totalValidators))
			return totalValidators, nil
		}

		// if not, keep finding out cosmos validators to collector all validators
		*pageAddress = page + 1
		if page > maxPage {
			return nil, errors.New("failed to find out all cosmos validators in this height")
		}
	}
}

func GetStakingValidatorsByHeight(c common.CommonClient, chainName string, height int64) ([]types.CosmosStakingValidator, error) {
	queryPath := types.CosmosStakingValidatorQueryPath(string(types.Bonded))
	stakingValidatorParser := parser.CosmosStakingValidatorParser

	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// set header by block height
	heightStr := strconv.FormatInt(height, 10)
	header := map[string]string{"Content-Type": "application/json", "x-cosmos-block-height": heightStr}
	c.APIClient.SetHeaders(header)

	// get on-chain validators in staking module
	resp, err := c.APIClient.Get(ctx, queryPath)
	if err != nil {
		// c.Errorf("api error: %s", err)
		return nil, errors.Wrap(err, "failed in api")
	}

	stakingValidators, err := stakingValidatorParser(resp)
	if err != nil {
		return nil, errors.Wrap(err, "failed in api")
	}

	// logging total validators count
	// c.Debugf("total cosmos staking validators: %d", len(stakingValidators))
	return stakingValidators, nil
}

// TODO: Move parsing logic into parser module for other blockchains
// TODO: first parameter should change from indexer struct to interface
// query current staking validators
func GetStakingValidators(c common.CommonClient, chainName string, status ...string) ([]types.CosmosStakingValidator, error) {
	var (
		defaultStatus          string = string(types.Bonded)
		queryPath              string
		stakingValidatorParser func(resp []byte) ([]types.CosmosStakingValidator, error)
	)

	if len(status) > 0 {
		defaultStatus = status[0]
	}

	switch chainName {
	case "initia":
		queryPath = types.InitiaStakingValidatorQueryPath(defaultStatus)
		stakingValidatorParser = parser.InitiaStakingValidatorParser
	case "story":
		queryPath = types.StoryStakingValidatorQueryPath(defaultStatus)
		stakingValidatorParser = parser.StoryStakingValidatorParser
	default:
		// NOTE: default query path will be cosmos-sdk staking module path
		queryPath = types.CosmosStakingValidatorQueryPath(defaultStatus)
		stakingValidatorParser = parser.CosmosStakingValidatorParser
	}

	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := c.APIClient

	// get on-chain validators in staking module
	resp, err := requester.Get(ctx, queryPath)
	if err != nil {
		// c.Errorf("api error: %s", err)
		return nil, errors.Wrap(err, "failed in api")
	}

	stakingValidators, err := stakingValidatorParser(resp)
	if err != nil {
		return nil, errors.Wrap(err, "failed in api")
	}

	// logging total validators count
	// c.Debugf("total cosmos staking validators: %d", len(stakingValidators))
	return stakingValidators, nil
}

func GetProviderValidators(c common.CommonClient, consumerID string) ([]types.ProviderValidator, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient
	resp, err := requester.Get(ctx, types.ProviderValidatorsQueryPath(consumerID))
	if err != nil {
		return nil, errors.Cause(err)
	}

	var result types.CosmosProviderValidatorsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, errors.Cause(err)
	}

	return result.Validators, nil
}

func GetConsumerChainID(c common.CommonClient) ([]types.ConsumerChain, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient
	resp, err := requester.Get(ctx, types.ConsumerChainListQueryPath)
	if err != nil {
		return nil, err
	}

	var result types.CosmosConsumerChainsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return result.Chains, nil
}

func GetConsumerChainHRP(c common.CommonClient) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient
	resp, err := requester.Get(ctx, types.CosmosSlashingLimitQueryPath)
	if err != nil {
		return "", errors.Cause(err)
	}

	var result types.CosmosSlashingResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", errors.Cause(err)
	}

	var hrp string
	for idx, info := range result.Info {
		valconsPrefix, _, err := sdkhelper.DecodeAndConvert(info.ConsensusAddress)
		if err != nil {
			return "", err
		}
		if idx == 0 {
			hrp = valconsPrefix
			break
		}
	}

	return hrp, nil
}

// query a new block to find missed validators index
func GetBlockResults(c common.CommonClient, height int64) (
	/* txs events */ []types.BlockEvent,
	/* block events */ []types.BlockEvent,
	/* unexpected error */ error,
) {

	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	resp, err := c.RPCClient.Get(ctx, types.CosmosBlockResultsQueryPath(height))
	if err != nil {
		return nil, nil, errors.Errorf("rpc call is failed from %s: %s", types.CosmosBlockResultsQueryPath(height), err)
	}

	txsEvents, blockEvents, err := parser.CosmosBlockResultsParser(resp)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return txsEvents, blockEvents, nil
}
