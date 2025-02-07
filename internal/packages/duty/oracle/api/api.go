package api

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/duty/oracle/types"
)

func GetOracleStatus(
	c *common.Exporter,
	CommonOracleQueryPath string,
	CommonOracleParser func([]byte) (uint64, error),
	CommonOracleParamsQueryPath string,
	CommonOracleParamsParser func([]byte) (slashWindow, votePeriod, minValidPerWindow, voteWindow float64, err error),
) (types.CommonOracleStatus, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := c.APIClient

	// get on-chain validators
	resp, err := requester.Get(ctx, types.CommonValidatorQueryPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonOracleStatus{}, common.ErrFailedHttpRequest
	}

	// json unmarsharling received validators data
	var validators types.CommonValidatorsQueryResponse
	if err := json.Unmarshal(resp, &validators); err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonOracleStatus{}, common.ErrFailedJsonUnmarshal
	}

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	missCountResults := make([]types.ValidatorStatus, 0)

	// add wg by the number of total validators
	wg.Add(len(validators.Validators))

	// get miss count by each validator
	for _, item := range validators.Validators {
		// set query path
		validatorOperatorAddress := item.OperatorAddress
		validatorMoniker := item.Description.Moniker
		queryPath := strings.Replace(CommonOracleQueryPath, "{validator_address}", validatorOperatorAddress, -1)

		// start go-routine
		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(c.Entry)
			defer wg.Done()

			resp, err := requester.Get(ctx, queryPath)
			if err != nil {
				if resp == nil {
					c.Errorln("[panic] passed resp.Time() nil point err")
					ch <- helper.Result{Item: nil, Success: false}
					return
				}
				c.Errorf("api error: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			missCount, err := CommonOracleParser(resp)
			if err != nil {
				c.Errorf("api error: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			ch <- helper.Result{
				Success: true,
				Item: types.ValidatorStatus{
					MissCounter:              missCount,
					ValidatorOperatorAddress: validatorOperatorAddress,
					Moniker:                  validatorMoniker,
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
			missCountResults = append(missCountResults, r.Item.(types.ValidatorStatus))
			continue
		}
		errorCount++
	}

	if errorCount > 0 {
		c.Errorf("failed to collect all miss count results from node, got errors count: %d", errorCount)
		return types.CommonOracleStatus{}, common.ErrFailedHttpRequest
	}

	// get oracle params by each chain
	resp, err = requester.Get(ctx, CommonOracleParamsQueryPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonOracleStatus{}, common.ErrFailedHttpRequest
	}

	slashWindow, votePeriod, minValidPerWindow, voteWindow, err := CommonOracleParamsParser(resp)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonOracleStatus{}, common.ErrFailedJsonUnmarshal
	}

	c.Debugf("metrics length: validators: %d, results: %d", len(validators.Validators), len(missCountResults))
	// TODO: add some validate logic
	if len(validators.Validators) != len(missCountResults) {
		c.Errorf("unmatched metrics length: validators: %d, but results: %d", len(validators.Validators), len(missCountResults))
	}

	// get latest height by each chain
	resp, err = requester.Get(ctx, types.CommonBlockHeightQueryPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonOracleStatus{}, common.ErrFailedHttpRequest
	}

	// json unmarsharling
	var block types.CommonLatestBlockQueryResponse
	if err := json.Unmarshal(resp, &block); err != nil {
		c.Errorf("parser error: %s", err)
		return types.CommonOracleStatus{}, common.ErrFailedJsonUnmarshal
	}

	blockHeight, err := strconv.ParseFloat(block.Block.Header.Height, 64)
	if err != nil {
		return types.CommonOracleStatus{}, err
	}

	return types.CommonOracleStatus{
		BlockHeight:           blockHeight,
		SlashWindow:           slashWindow,
		MinimumValidPerWindow: minValidPerWindow,
		VotePeriod:            votePeriod,
		VoteWindow:            voteWindow,
		Validators:            missCountResults,
	}, nil
}
