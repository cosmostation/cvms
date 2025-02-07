package api

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/parser"
	"github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/helper"
	fputypes "github.com/cosmostation/cvms/internal/packages/duty/finality-provider-uptime/types"
	"github.com/pkg/errors"
)

func GetActiveFinalityProviderByHeight(c common.CommonClient, height int64) ([]types.FinalityProvider, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	resp, err := c.APIClient.Get(ctx, types.BabylonFinalityProvidersQueryPath(height))
	if err != nil {
		return nil, errors.Errorf("rpc call is failed from %s: %s", types.BabylonFinalityProvidersQueryPath(height), err)
	}

	fps, err := parser.ParseFinalityProviders(resp)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return fps.FinalityProviders, nil
}

func GetFinalityVotesByHeight(c common.CommonClient, height int64) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	resp, err := c.APIClient.Get(ctx, types.BabylonFinalityVotesQueryPath(height))
	if err != nil {
		return nil, errors.Errorf("rpc call is failed from %s: %s", types.BabylonFinalityVotesQueryPath(height), err)
	}

	votes, err := parser.ParseFinalityProviderVotings(resp)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return votes.BTCPKs, nil
}

func GetBabylonFinalityProviderInfos(c common.CommonClient) ([]types.FinalityProviderInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	fpInfoList := make([]types.FinalityProviderInfo, 0)
	maxCnt := 10
	key := ""
	for cnt := 0; cnt <= maxCnt; cnt++ {
		resp, err := c.APIClient.Get(ctx, types.BabylonFinalityProviderInfosQueryPath(key))
		if err != nil {
			return nil, errors.Errorf("rpc call is failed from %s: %s", types.BabylonFinalityProviderInfosQueryPath(key), err)
		}

		fpInfos, err := parser.ParseFinalityProviderInfos(resp)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		fpInfoList = append(fpInfoList, fpInfos.FinalityProviders...)

		if fpInfos.Pagination.NextKey != "" {
			key = url.QueryEscape(fpInfos.Pagination.NextKey)
			c.Debugf("there is next key, keep collecting finality providers by using this path: %s", types.BabylonFinalityProviderInfosQueryPath(key))
		} else {
			// got all finality provider infos
			c.Debugf("collected all finality providers")
			break
		}
	}

	return fpInfoList, nil
}

func GetFinalityProviderUptime(c common.CommonClient, fpInfoList []types.FinalityProviderInfo) ([]fputypes.FinalityProviderUptimeStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	validatorResult := make([]fputypes.FinalityProviderUptimeStatus, 0)
	wg.Add(len(fpInfoList))
	for _, item := range fpInfoList {
		moniker := item.Description.Moniker
		Address := item.Address
		BTCPK := item.BTCPK
		queryPath := types.BabylonFinalityProviderSigninInfoQueryPath(item.BTCPK)
		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(c.Entry)
			defer wg.Done()

			resp, err := c.APIClient.Get(ctx, queryPath)
			if err != nil {
				if resp == nil {
					c.Errorln("unexpected nil response")
					ch <- helper.Result{Item: nil, Success: false}
					return
				}
				c.Errorf("unexpected err: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			missedBlockCounter, err := parser.ParserFinalityProviderSigningInfo(resp)
			if err != nil {
				c.Errorf("unexpected err: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			ch <- helper.Result{
				Success: true,
				Item: fputypes.FinalityProviderUptimeStatus{
					Moniker:            moniker,
					Address:            Address,
					BTCPK:              BTCPK,
					MissedBlockCounter: missedBlockCounter,
				}}
		}(ch)
		time.Sleep(10 * time.Millisecond)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	errorCount := 0
	for r := range ch {
		if r.Success {
			validatorResult = append(validatorResult, r.Item.(fputypes.FinalityProviderUptimeStatus))
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

func GetBabylonFinalityProviderParams(c common.CommonClient) (float64, float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	resp, err := c.APIClient.Get(ctx, types.BabylonFinalityParamsQueryPath)
	if err != nil {
		return 0, 0, errors.Errorf("rpc call is failed from %s: %s", types.BabylonFinalityParamsQueryPath, err)
	}

	signedBlocksWindow, minSignedPerWindow, err := parser.ParserFinalityParams(resp)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	return signedBlocksWindow, minSignedPerWindow, nil
}

func GetBabylonBTCLightClientParams(c common.CommonClient) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient
	resp, err := requester.Get(ctx, types.BabylonBTCLightClientParamsQueryPath)
	if err != nil {
		endpoint, _ := requester.GetEndpoint()
		return nil, errors.Errorf("rpc call is failed from %s: %s", endpoint, err)
	}

	allowList, err := parser.ParserBTCLightClientParams(resp)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return allowList, nil
}
