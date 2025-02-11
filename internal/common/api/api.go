package api

import (
	"context"
	"encoding/json"
	"net/http"
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

	requester := c.RPCClient.R().SetContext(ctx)
	resp, err := requester.Get(types.CosmosStatusQueryPath)
	if err != nil {
		return 0, time.Time{}, errors.Errorf("rpc call is failed from %s: %s", resp.Request.URL, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return 0, time.Time{}, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
	}

	latestBlockHeight, latestBlockTimestamp, err := parser.CosmosStatusParser(resp.Body())
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
	requester := c.RPCClient.R().SetContext(ctx)

	resp, err := requester.Get(types.CosmosBlockQueryPath(height))
	if err != nil {
		return 0, time.Time{}, "", nil, 0, nil, errors.Errorf("rpc call is failed from %s: %s", resp.Request.URL, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return 0, time.Time{}, "", nil, 0, nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
	}

	blockHeight, blockTimeStamp, blockProposerAddress, blockTxs, lastCommitBlockHeight, blockSignatures, err := parser.CosmosBlockParser(resp.Body())
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

	// create requester
	requester := c.RPCClient.R().SetContext(ctx)

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

		resp, err := requester.Get(queryPath)
		if err != nil {
			return nil, errors.Errorf("rpc call is failed from %s: %s", resp.Request.URL, err)
		}
		if resp.StatusCode() != http.StatusOK {
			return nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
		}

		validators, totalValidatorsCount, err := parser.CosmosValidatorParser(resp.Body())
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

	// create requester
	requester := c.APIClient.R().SetContext(ctx)

	// set header by block height
	heightStr := strconv.FormatInt(height, 10)
	header := map[string]string{"Content-Type": "application/json", "x-cosmos-block-height": heightStr}
	requester.SetHeaders(header)

	// get on-chain validators in staking module
	resp, err := requester.Get(queryPath)
	if err != nil {
		// c.Errorf("api error: %s", err)
		return nil, errors.Wrap(err, "failed in api")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}

	stakingValidators, err := stakingValidatorParser(resp.Body())
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
	requester := c.APIClient.R().SetContext(ctx)

	// get on-chain validators in staking module
	resp, err := requester.Get(queryPath)
	if err != nil {
		// c.Errorf("api error: %s", err)
		return nil, errors.Wrap(err, "failed in api")
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}

	stakingValidators, err := stakingValidatorParser(resp.Body())
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

	requester := c.APIClient.R().SetContext(ctx)
	resp, err := requester.Get(types.ProviderValidatorsQueryPath(consumerID))
	if err != nil {
		return nil, errors.Cause(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Wrapf(err, "api error: got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}

	var result types.CosmosProviderValidatorsResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, errors.Cause(err)
	}

	return result.Validators, nil
}

func GetConsumerChainID(c common.CommonClient) ([]types.ConsumerChain, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient.R().SetContext(ctx)
	resp, err := requester.Get(types.ConsumerChainListQueryPath)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("api error: got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}

	var result types.CosmosConsumerChainsResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, err
	}

	return result.Chains, nil
}

func GetConsumerChainHRP(c common.CommonClient) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient.R().SetContext(ctx)
	resp, err := requester.Get(types.CosmosSlashingLimitQueryPath)
	if err != nil {
		return "", errors.Cause(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return "", errors.Wrapf(err, "api error: got %d code from %s", resp.StatusCode(), resp.Request.URL)
	}

	var result types.CosmosSlashingResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
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

	// create requester
	requester := c.RPCClient.R().SetContext(ctx)

	resp, err := requester.Get(types.CosmosBlockResultsQueryPath(height))
	if err != nil {
		return nil, nil, errors.Errorf("rpc call is failed from %s: %s", resp.Request.URL, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())

	}

	txsEvents, blockEvents, err := parser.CosmosBlockResultsParser(resp.Body())
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return txsEvents, blockEvents, nil
}

// query block and txs data by using cosmos api endpoint
func GetBlockAndTxs(c common.CommonClient, height int64) (
	int64,
	time.Time,
	[]types.CosmosTx,
	error,
) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient.R().SetContext(ctx)
	resp, err := requester.Get(types.CosmosBlockTxsQueryPath(height))
	if err != nil {
		return 0, time.Time{}, nil, errors.Errorf("rpc call is failed from %s: %s", resp.Request.URL, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return 0, time.Time{}, nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
	}

	var result types.CosmosBlockAndTxsResponse
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return 0, time.Time{}, nil, errors.Wrap(err, "failed to unmarshal response")
	}

	if result.Pagination.NextKey != "" {
		return 0, time.Time{}, nil, errors.New("pagination is not supported yet")
	}

	blockHeight, err := strconv.ParseInt(result.Block.Header.Height, 10, 64)
	if err != nil {
		return 0, time.Time{}, nil, errors.Wrap(err, "failed to parse block height")
	}

	return blockHeight, result.Block.Header.Time, result.Txs, nil
}
