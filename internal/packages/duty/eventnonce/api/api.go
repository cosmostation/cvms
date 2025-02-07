package api

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/duty/eventnonce/types"
)

func GetEventNonceStatus(
	c *common.Exporter,
	CommonOrchestratorPath string,
	CommonOrchestratorParser func([]byte) (string, error),
	CommonEventNonceQueryPath string,
	CommonEventNonceParser func([]byte) (float64, error),
) (types.CommonEventNonceStatus, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := c.APIClient

	// get on-chain validators
	resp, err := requester.Get(ctx, types.CommonValidatorQueryPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonEventNonceStatus{}, common.ErrFailedHttpRequest
	}

	// json unmarsharling received validators data
	var validators types.CommonValidatorsQueryResponse
	if err := json.Unmarshal(resp, &validators); err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonEventNonceStatus{}, common.ErrFailedJsonUnmarshal
	}

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	validatorResults := make([]types.ValidatorStatus, 0)

	// add wg by the number of total validators
	wg.Add(len(validators.Validators))

	// get validator's orchestrator address
	for _, item := range validators.Validators {
		// set query path
		validatorOperatorAddress := item.OperatorAddress
		validatorMoniker := item.Description.Moniker
		queryPath := CommonOrchestratorPath + "?validator_address=" + validatorOperatorAddress

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

			orchestratorAddress, err := CommonOrchestratorParser(resp)
			if err != nil {
				c.Errorf("api error: %s", err)
				ch <- helper.Result{Success: false, Item: nil}
				return
			}

			ch <- helper.Result{
				Success: true,
				Item: types.ValidatorStatus{
					ValidatorOperatorAddress: validatorOperatorAddress,
					OrchestratorAddress:      orchestratorAddress,
					Moniker:                  validatorMoniker,
				},
			}
		}(ch)

		time.Sleep(10 * time.Millisecond)
	}

	// close channel
	go func() {
		wg.Wait()
		close(ch)
	}()

	// collect validator's orch
	for r := range ch {
		if r.Success {
			validatorResults = append(validatorResults, r.Item.(types.ValidatorStatus))
			continue
		}
		// errorCount++
	}

	// if errorCount > 0 {
	// 	c.Errorf("failed to collect all validator results from node, got errors count: %d", errorCount)
	// 	return types.CommonEventNonceStatus{}, common.ErrFailedHttpRequest
	// }

	if len(validators.Validators) != len(validatorResults) {
		c.Warnf("total validators: %d, but total orchestrators: %d", len(validators.Validators), len(validatorResults))
		// example: // https://gravity-api.polkachu.com/gravity/v1beta/query_delegate_keys_by_validator?validator_address=gravityvaloper14m0n559xdj00qwvp6ck0xesprrq26kgp909wm2
		c.Warnf("because some of validators didn't register their own orchesrator address in the chain")
	}

	// init channel and waitgroup for go-routine
	ch = make(chan helper.Result)
	eventNonceResults := make([]types.ValidatorStatus, 0)

	// add wg by the number of total orchestrators
	wg = sync.WaitGroup{}
	wg.Add(len(validatorResults))

	// get eventnonce by each orchestrator
	for _, item := range validatorResults {
		validatorOperatorAddress := item.ValidatorOperatorAddress
		orchestratorAddress := item.OrchestratorAddress
		validatorMoniker := item.Moniker
		queryPath := strings.Replace(CommonEventNonceQueryPath, "{orchestrator_address}", orchestratorAddress, -1)

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

			eventNonce, err := CommonEventNonceParser(resp)
			if err != nil {
				c.Errorf("api error: %s", err)
				ch <- helper.Result{Success: false, Item: nil}
				return
			}

			ch <- helper.Result{
				Success: true,
				Item: types.ValidatorStatus{
					Moniker:                  validatorMoniker,
					ValidatorOperatorAddress: validatorOperatorAddress,
					OrchestratorAddress:      orchestratorAddress,
					EventNonce:               eventNonce,
				},
			}
		}(ch)
		time.Sleep(10 * time.Millisecond)
	}

	// close channels
	go func() {
		wg.Wait()
		close(ch)
	}()

	// collect results
	errorCounter := 0
	for r := range ch {
		if r.Success {
			eventNonceResults = append(eventNonceResults, r.Item.(types.ValidatorStatus))
			continue
		}
		errorCounter++
	}

	// valset is not same between gbt set and staking validators set
	if errorCounter > 0 {
		// https://gravity-api.polkachu.com/gravity/v1beta/valset/current
		// >> 117

		// https://gravity-api.polkachu.com/cosmos/staking/v1beta1/validators?pagination.limit=500&status=BOND_STATUS_BONDED
		// >> 120

		// return types.CommonEventNonceStatus{}, fmt.Errorf("unexpected errors was found: total %d errors", errorCounter)
		c.Warnf("some of validators didn't set own gravity orchestrator address at gbt module: %d", errorCounter)
	}

	// find heighest eventnonce in the results
	heighestEventNonce := float64(0)
	for idx, item := range eventNonceResults {
		if idx == 0 {
			heighestEventNonce = item.EventNonce
			c.Debugln("set heighest nonce:", heighestEventNonce)
		}

		if item.EventNonce > heighestEventNonce {
			c.Debugln("changed heightest nonce from: ", heighestEventNonce, "to: ", item.EventNonce)
			heighestEventNonce = item.EventNonce
		}
	}

	return types.CommonEventNonceStatus{
		HeighestNonce: heighestEventNonce,
		Validators:    eventNonceResults,
	}, nil
}
