package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/duty/eventnonce/types"
)

// NOTE: debug
// func init() {
// 	logger := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr)
// 	grpclog.SetLoggerV2(logger)
// }

func GetEventNonceStatusByGRPC(
	c *common.Exporter,
	commonOrchestratorPath string, commonOrchestratorParser func([]byte) (string, error),
	commonEventNonceQueryPath string, commonEventNonceParser func([]byte) (float64, error),
) (types.CommonEventNonceStatus, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	resp, err := c.GRPCClient.Post(ctx, types.CommonValidatorGrpcQueryPath, []byte(types.CommonValidatorGrpcQueryOption))
	if err != nil {
		c.Errorf("grpc request err: %s", err.Error())
		return types.CommonEventNonceStatus{}, common.ErrFailedGrpcRequest
	}

	// json unmarsharling received validators data
	var validators types.CommonValidatorsQueryResponse
	if err := json.Unmarshal(resp, &validators); err != nil {
		c.Errorf("parser error: %s", err)
		return types.CommonEventNonceStatus{}, common.ErrFailedJsonUnmarshal
	}

	// init channel and waitgroup
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	// var methodDescriptor protoreflect.MethodDescriptor
	validatorResults := make([]types.ValidatorStatus, 0)

	// add wg by the number of total validators
	wg.Add(len(validators.Validators))

	// get validators orchestrator address
	for _, item := range validators.Validators {
		// in only first time, make a method descriptor by using grpc reflection client
		// if idx == 0 {
		// 	methodDescriptor, err = grpchelper.GrpcMakeDescriptor(
		// 		reflectionClient,       // grpc reflection client
		// 		commonOrchestratorPath, // grpc reflection method path
		// 	)
		// 	if err != nil {
		// 		c.Errorln("grpc api err: failed to make method descprtior")
		// 		return types.CommonEventNonceStatus{}, common.ErrFailedGrpcRequest
		// 	}
		// }

		validatorOperatorAddress := item.OperatorAddress
		validatorMoniker := item.Description.Moniker
		commonOrchestratorPayload := fmt.Sprintf(`{"validator_address":"%s"}`, validatorOperatorAddress)

		go func(ch chan helper.Result) {
			defer wg.Done()

			resp, err := c.GRPCClient.Post(ctx, commonOrchestratorPath, []byte(commonOrchestratorPayload))
			if err != nil {
				c.Errorf("grpc error: %s", err)
				ch <- helper.Result{Success: false, Item: nil}
				return
			}

			orchestratorAddress, err := commonOrchestratorParser(resp)
			if err != nil {
				c.Errorf("grpc error: %v", err)
				ch <- helper.Result{Success: false, Item: nil}
				return
			}

			if orchestratorAddress == "" {
				// not registered validators
				c.Warnf("got empty orchestrator address for %s, so saved empty string", validatorOperatorAddress)
				ch <- helper.Result{Success: true, Item: types.ValidatorStatus{
					ValidatorOperatorAddress: validatorOperatorAddress,
					OrchestratorAddress:      "",
					Moniker:                  validatorMoniker,
				}}
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
	errorCounter := 0
	for r := range ch {
		if r.Success {
			validatorResults = append(validatorResults, r.Item.(types.ValidatorStatus))
			continue
		}
		errorCounter++
	}

	if errorCounter > 0 {
		return types.CommonEventNonceStatus{}, fmt.Errorf("unexpected errors was found: total %d errors", errorCounter)
	}

	c.Debugf("total validators: %d, total orchestrator result len: %d", len(validators.Validators), len(validatorResults))

	// init channel and waitgroup for go-routine
	ch = make(chan helper.Result)
	eventNonceResults := make([]types.ValidatorStatus, 0)

	// add wg by the number of total orchestrators
	wg = sync.WaitGroup{}
	wg.Add(len(validatorResults))

	// get eventnonce by each orchestrator
	for _, item := range validatorResults {
		// if idx == 0 {
		// 	methodDescriptor, err = grpchelper.GrpcMakeDescriptor(
		// 		reflectionClient,          // grpc reflection client
		// 		commonEventNonceQueryPath, // grpc reflection method path
		// 	)
		// 	if err != nil {
		// 		c.Errorln("grpc api err: failed to make method descprtior")
		// 		return types.CommonEventNonceStatus{}, common.ErrFailedGrpcRequest
		// 	}
		// }

		orchestratorAddress := item.OrchestratorAddress
		validatorOperatorAddress := item.ValidatorOperatorAddress
		validatorMoniker := item.Moniker
		payload := fmt.Sprintf(`{"address":"%s"}`, orchestratorAddress)

		go func(ch chan helper.Result) {
			defer wg.Done()
			if orchestratorAddress == "" {
				c.Warnf("skipped empty orchestrator address for %s", validatorOperatorAddress)
				ch <- helper.Result{
					Success: true,
					Item: types.ValidatorStatus{
						ValidatorOperatorAddress: validatorOperatorAddress,
						OrchestratorAddress:      orchestratorAddress,
						EventNonce:               0,
					},
				}
				return
			}

			resp, err := c.GRPCClient.Post(ctx, commonEventNonceQueryPath, []byte(payload))
			if err != nil {
				c.Errorf("grpc error: %s", err)
				ch <- helper.Result{Success: false, Item: nil}
				return
			}

			eventNonce, err := commonEventNonceParser([]byte(resp))
			if err != nil {
				c.Errorf("grpc error: %v", err)
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
	errorCounter = 0
	for r := range ch {
		if r.Success {
			eventNonceResults = append(eventNonceResults, r.Item.(types.ValidatorStatus))
			continue
		}
		errorCounter++
	}

	if errorCounter > 0 {
		return types.CommonEventNonceStatus{}, fmt.Errorf("unexpected errors was found: total %d errors", errorCounter)
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
