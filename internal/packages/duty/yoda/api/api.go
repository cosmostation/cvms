package api

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	commonapi "github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/duty/yoda/types"
)

func GetBlockHeight(
	c *common.Exporter,
	CommonYodaBlockHeightRequestPath string,
	CommonYodaBlockHeightResponseParser func([]byte) (blockHeight int64, err error),
) (int64, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := c.APIClient
	// get current blockheight
	resp, err := requester.Get(ctx, CommonYodaBlockHeightRequestPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return 0, common.ErrFailedHttpRequest
	}

	blockHeight, err := CommonYodaBlockHeightResponseParser(resp)
	if err != nil {
		c.Errorf("api error: %s", err)
		return 0, common.ErrFailedJsonUnmarshal
	}
	return blockHeight, nil
}

func GetYodaStatus(
	c *common.Exporter,
	CommonYodaQueryPath string,
	CommonYodaParser func([]byte) (float64, error),
	CommonYodaParamsPath string,
	CommonYodaParamsParser func([]byte) (slashWindow float64, err error),
	CommonYodaRequestCountsPath string,
	CommonYodaRequestCountsParser func([]byte) (requestCount float64, err error),
	CommonYodaRequestPath string,
	CommonYodaRequestParser func([]byte) (requestBlock int64, validatorsFailedToRespond []string, status string, err error),
) (types.CommonYodaStatus, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := c.APIClient

	// get on-chain validators
	resp, err := requester.Get(ctx, types.CommonValidatorQueryPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonYodaStatus{}, common.ErrFailedHttpRequest
	}

	// json unmarsharling received validators data
	var validators types.CommonValidatorsQueryResponse
	if err := json.Unmarshal(resp, &validators); err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonYodaStatus{}, common.ErrFailedJsonUnmarshal
	}

	// get current request count
	resp, err = requester.Get(ctx, CommonYodaRequestCountsPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonYodaStatus{}, common.ErrFailedHttpRequest
	}

	// json unmarsharling received request count data
	requestCount, err := CommonYodaRequestCountsParser(resp)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonYodaStatus{}, common.ErrFailedJsonUnmarshal
	}
	c.Debugf("Current total request count: %.2f", requestCount)

	// get oracle params by each chain
	resp, err = requester.Get(ctx, CommonYodaParamsPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonYodaStatus{}, common.ErrFailedHttpRequest
	}

	yodaSlashWindow, err := CommonYodaParamsParser(resp)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonYodaStatus{}, common.ErrFailedJsonUnmarshal
	}

	currentBlockHeight, _, err := commonapi.GetStatus(c.CommonClient)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonYodaStatus{}, common.ErrFailedHttpRequest
	}

	//
	// Get non expired requests
	//
	nonExpiredResults := make([]types.RequestStatus, 0)

	// As we don't want to query all requests all the time we run this in
	// batches to reduce the number of requests sent to the API.
	// If a batch contains an expired request, the loop will break as
	// all other requests in the following batches will be expired as well.
	var batchSize int = 20

	var expiredCount int = 0
	for i := int(requestCount); i > 1; i -= batchSize {
		// init channel and waitgroup for go-routine
		nerCh := make(chan helper.Result)
		var nerWg sync.WaitGroup
		for request := i; request > i-batchSize; request-- {
			// add new wg member
			nerWg.Add(1)

			// set query path
			queryPath := strings.Replace(CommonYodaRequestPath, "{request_id}", strconv.Itoa(request), -1)

			// start go-routine
			go func(nerCh chan helper.Result) {
				defer helper.HandleOutOfNilResponse(c.Entry)
				defer nerWg.Done()

				resp, err := requester.Get(ctx, queryPath)
				if err != nil {
					if resp == nil {
						c.Errorln("[panic] passed resp.Time() nil point err")
						nerCh <- helper.Result{Item: nil, Success: false}
						return
					}
					c.Errorf("api error: %s", err)
					nerCh <- helper.Result{Item: nil, Success: false}
					return
				}

				requestBlock, validatorsFailedToRespond, status, err := CommonYodaRequestParser(resp)
				if err != nil {
					c.Errorf("api error: %s", err)
					nerCh <- helper.Result{Item: nil, Success: false}
					return
				}
				nerCh <- helper.Result{Success: true, Item: types.RequestStatus{
					RequestID:                 int64(request),
					Status:                    status,
					RequestHeight:             requestBlock,
					BlocksPassed:              currentBlockHeight - requestBlock,
					ValidatorsFailedToRespond: validatorsFailedToRespond,
				}}
			}(nerCh)

		}
		// close channel when all go-routines are done
		go func() {
			nerWg.Wait()
			close(nerCh)
		}()

		for nerResult := range nerCh {
			if nerResult.Success {
				if nerResult.Item.(types.RequestStatus).RequestHeight+int64(yodaSlashWindow) < currentBlockHeight {
					expiredCount++
					continue
				} else if nerResult.Item.(types.RequestStatus).BlocksPassed >= 0 {
					nonExpiredResults = append(nonExpiredResults, nerResult.Item.(types.RequestStatus))
				}
			}
		}
		if expiredCount > 0 {
			break
		}
	}
	c.Debugf("Found non-expired requests: %d", len(nonExpiredResults))

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	yodaResults := make([]types.ValidatorStatus, 0)

	// add wg by the number of total validators
	wg.Add(len(validators.Validators))

	// get miss count by each validator
	for _, item := range validators.Validators {
		// set query path
		validatorOperatorAddress := item.OperatorAddress
		validatorMoniker := item.Description.Moniker
		queryPath := strings.Replace(CommonYodaQueryPath, "{validator_address}", validatorOperatorAddress, -1)

		// map missed requests to validator
		var maxMisses float64 = 0
		validatorFailedRequests := make([]types.RequestStatus, 0)
		for _, request := range nonExpiredResults {
			if slices.Contains(request.ValidatorsFailedToRespond, item.OperatorAddress) {
				maxMisses = math.Max(maxMisses, float64(request.BlocksPassed))
				validatorFailedRequests = append(validatorFailedRequests, request)
			}
		}

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

			isActive, err := CommonYodaParser(resp)
			if err != nil {
				c.Errorf("api error: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			ch <- helper.Result{
				Success: true,
				Item: types.ValidatorStatus{
					IsActive:                 isActive,
					ValidatorOperatorAddress: validatorOperatorAddress,
					Moniker:                  validatorMoniker,
					MaxMisses:                maxMisses,
					Requests:                 validatorFailedRequests,
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
			yodaResults = append(yodaResults, r.Item.(types.ValidatorStatus))
			continue
		}
		errorCount++
	}

	if errorCount > 0 {
		c.Errorf("failed to collect all validator results from node, got errors count: %d", errorCount)
		return types.CommonYodaStatus{}, common.ErrFailedHttpRequest
	}

	return types.CommonYodaStatus{
		SlashWindow:  yodaSlashWindow,
		RequestCount: requestCount,
		Validators:   yodaResults,
	}, nil
}
