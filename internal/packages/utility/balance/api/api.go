package api

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper"
	balanceErrors "github.com/cosmostation/cvms/internal/packages/utility/balance/errors"
	"github.com/cosmostation/cvms/internal/packages/utility/balance/types"
)

func GetBalanceStatus(
	c *common.Exporter,
	CommonBalanceCallMethod common.Method,
	CommonBalanceQueryPath string,
	CommonBalancePayload string,
	CommonBalanceParser func([]byte, string) (float64, error),
	BalanceAddresses []string, BalanceDenom string, BalanceExponent int,
) (types.CommonBalance, error) {
	// init context
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	// create requester
	requester := c.APIClient

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup
	totalStatus := make([]types.BalanceStatus, 0)

	wg.Add(len(BalanceAddresses))

	// get evm status by each validator
	for _, address := range BalanceAddresses {
		var resp = []byte{}
		var err error

		address := address
		payload := strings.Replace(CommonBalancePayload, "{balance_address}", address, -1)
		queryPath := strings.Replace(CommonBalanceQueryPath, "{balance_address}", address, -1)

		// start go-routine
		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(c.Entry)
			defer wg.Done()

			if CommonBalanceCallMethod == common.GET {
				resp, err = requester.Get(ctx, queryPath)
			} else {
				resp, err = requester.Post(ctx, queryPath, []byte(payload))
			}

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

			remainingBalance, err := CommonBalanceParser(resp, BalanceDenom)
			// catch expected error when balance is not found and assuming balance as 0
			if errors.Is(err, balanceErrors.ErrBalanceNotFound) {
				c.Warningf("Balance not found for %s and denom %s, assuming balance as 0", address, BalanceDenom)
				remainingBalance = 0.0
			} else if err != nil {
				c.Errorf("api error: %s", err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// // calculate exponent with pow
			// exponentPower, err := strconv.Atoi(BalanceExponent)
			// if err != nil {
			// 	c.Errorf("api error: %s", err)
			// 	ch <- helper.Result{Item: nil, Success: false}
			// 	return
			// }

			// calculate remaining balance to look easily
			trimmedBalance := (remainingBalance / math.Pow10(BalanceExponent))
			c.Debugf("found remaining %s trimmed balance: %.2f in %s", BalanceDenom, trimmedBalance, address)

			ch <- helper.Result{
				Success: true,
				Item: types.BalanceStatus{
					Address:          address,
					RemainingBalance: trimmedBalance,
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

	// collect balance's status
	errorCount := 0
	for r := range ch {
		if r.Success {
			if item, ok := r.Item.(types.BalanceStatus); ok {
				totalStatus = append(totalStatus, item)
				continue
			}
		}
		errorCount++
	}

	if errorCount > 0 {
		c.Errorf("failed to collect all accounts balances, got errors count: %d", errorCount)
		return types.CommonBalance{}, common.ErrFailedHttpRequest
	}

	return types.CommonBalance{Balances: totalStatus}, nil
}
