package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/duty/axelar-evm/types"
)

func GetAxelarNexusStatus(
	exporter *common.Exporter,
	CommonEvmChainsQueryPath string,
	CommonEvmChainsParser func([]byte) ([]string, error),
	CommonEvmChainMaintainerQueryPath string,
	CommonChainMaintainersParser func([]byte) ([]string, error),
) (types.CommonAxelarNexus, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := exporter.APIClient

	// get on-chain validators
	resp, err := requester.Get(ctx, types.CommonValidatorQueryPath)
	if err != nil {
		exporter.Errorf("api error: %s", err)
		return types.CommonAxelarNexus{}, common.ErrFailedHttpRequest
	}

	// json unmarsharling received validators data
	var validators types.CommonValidatorsQueryResponse
	if err := json.Unmarshal(resp, &validators); err != nil {
		exporter.Errorf("api error: %s", err)
		return types.CommonAxelarNexus{}, common.ErrFailedJsonUnmarshal
	}

	// get on-chain active evm-chains
	resp, err = requester.Get(ctx, CommonEvmChainsQueryPath)
	if err != nil {
		exporter.Errorf("api error: %s", err)
		return types.CommonAxelarNexus{}, common.ErrFailedHttpRequest
	}

	activatedEvmChains, err := CommonEvmChainsParser(resp)
	if err != nil {
		return types.CommonAxelarNexus{}, err
	}
	exporter.Debugln("currently activated evm chains in axelar:", activatedEvmChains)

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	totalStatus := make([]types.ValidatorStatus, 0)

	// add wg by the number of active evm chains
	wg.Add(len(activatedEvmChains))

	// get evm status by each validator
	for _, evmChain := range activatedEvmChains {
		// set query path and variables
		queryPath := strings.Replace(CommonEvmChainMaintainerQueryPath, "{chain}", evmChain, -1)
		maintainerMap := make(map[string]float64)
		chainStatus := make([]types.ValidatorStatus, 0)
		chainName := evmChain

		// start go-routine
		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(exporter.Entry)
			defer wg.Done()

			resp, err = requester.Get(ctx, queryPath)
			if err != nil {
				if resp == nil {
					exporter.Errorln("[panic] passed resp.Time() nil point err")
					ch <- helper.Result{Item: nil, Success: false}
					return
				}
				exporter.Errorf("api error: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			maintainers, err := CommonChainMaintainersParser(resp)
			if err != nil {
				exporter.Errorf("api error: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			for _, maintainer := range maintainers {
				maintainerMap[maintainer] = 1
			}

			for _, item := range validators.Validators {
				chainStatus = append(chainStatus, types.ValidatorStatus{
					Moniker:                  item.Description.Moniker,
					ValidatorOperatorAddress: item.OperatorAddress,
					Status:                   maintainerMap[item.OperatorAddress],
					EVMChainName:             chainName,
				})
			}

			exporter.Debugf("total validators: %d and %s chain status results: %d", len(validators.Validators), chainName, len(chainStatus))
			ch <- helper.Result{
				Success: true,
				Item:    chainStatus,
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
	errorCounter := 0
	for r := range ch {
		if r.Success {
			if item, ok := r.Item.([]types.ValidatorStatus); ok {
				totalStatus = append(totalStatus, item...)
				continue
			}
		}
		errorCounter++
	}

	if errorCounter > 0 {
		return types.CommonAxelarNexus{}, errors.New("failed to get all validators status from go-routine")
	}

	return types.CommonAxelarNexus{
		ActiveEVMChains: activatedEvmChains,
		Validators:      totalStatus,
	}, nil
}
