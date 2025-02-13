package indexer

import (
	"fmt"
	"time"

	"sync"

	"github.com/cosmostation/cvms/internal/common/api"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-covenant-signature/model"
)

// NOTE: babylon covenant signature will be created at random block (When Delegation TX occurs)
// first we can get the block txs from the chain
// 1. get block txs /cosmos/tx/v1beta1/txs/block/{height}
// and then, query the first decoded tx in the block
// 2. filtering "/babylon.btcstaking.v1.MsgAddCovenantSigs" message in block txs
// 3. make a list for covenantSigs of committee members
func (idx *CovenantSignatureIndexer) batchSync(lastIndexPointerHeight, newIndexPointerHeight int64) (
	/* new index pointer */ int64,
	/* error */ error,
) {
	if lastIndexPointerHeight >= idx.Lh.LatestHeight {
		idx.Infof("current height is %d and latest height is %d both of them are same, so it'll skip the logic", lastIndexPointerHeight, idx.Lh.LatestHeight)
		return lastIndexPointerHeight, nil
	}

	// set starntHeight and endHeight for batch sync
	startHeight := newIndexPointerHeight
	endHeight := idx.Lh.LatestHeight

	// set limit at end-height in this batch sync logic
	if (idx.Lh.LatestHeight - newIndexPointerHeight) > indexertypes.BatchSyncLimit {
		endHeight = newIndexPointerHeight + indexertypes.BatchSyncLimit
		idx.Debugf("by batch sync limit, end height will change to %d", endHeight)
	}

	// init channel and waitgroup for go-routine
	ch1 := make(chan helper.Result)
	ch2 := make(chan helper.Result)
	var wg sync.WaitGroup

	// init covenant signature list
	covenantSignatureList := make([]model.BabylonCovenantSignature, 0)
	btcDelegationsList := make([]model.BabylonBtcDelegation, 0)

	// This timestamp for metrics
	var endBlockTimestamp time.Time

	_ = covenantSignatureList
	for height := startHeight; height <= endHeight; height++ {
		wg.Add(1)
		height := height

		go func(ch1 chan helper.Result, ch2 chan helper.Result) {
			defer helper.HandleOutOfNilResponse(idx.Entry)
			defer wg.Done()

			backoffTime := time.Second * 1

		RETRY:
			blockHeight, blockTimestamp, txs, err := api.GetBlockAndTxs(idx.CommonClient, height)
			if len(txs) <= 0 {
				return
			}

			if height == endHeight {
				endBlockTimestamp = blockTimestamp
			}

			covenantSigs, createBtcDelegations, err := ExtractBabylonCovenantSignature(txs)
			if err != nil {
				idx.Errorf("failed to extract resp.Body(), %s", err)
				helper.ExponentialBackoff(&backoffTime)
				goto RETRY
			}

			var newBtcDelegations = make([]model.BabylonBtcDelegation, 0)

			for _, delegateMsg := range createBtcDelegations {
				btcStakingTxHash, err := DecodeBtcStakingTx(delegateMsg.StakingTx)
				if err != nil {
					idx.Errorf("failed to decode btc staking tx, %s", err)
					helper.ExponentialBackoff(&backoffTime)
					goto RETRY
				}

				newBtcDelegation := model.BabylonBtcDelegation{
					ChainInfoID:      idx.ChainInfoID,
					Height:           blockHeight,
					BTCStakingTxHash: btcStakingTxHash,
					Timestamp:        blockTimestamp,
				}

				newBtcDelegations = append(newBtcDelegations, newBtcDelegation)
			}

			ch1 <- helper.Result{
				Item:    newBtcDelegations,
				Success: true,
			}

			var newBcsList = make([]model.BabylonCovenantSignature, 0)

			for _, sig := range covenantSigs {

				// It's not yet clear if Committee members can change dynamically, we've added some temporary code to prevent panic
				pkID, exists := idx.covenantCommitteeMap[sig.Pk]
				if !exists {
					idx.Errorf("Missing covenant committee entry for PK: %s", sig.Pk)
					continue
				}

				newCovenantSignature := model.BabylonCovenantSignature{
					ChainInfoID:      idx.ChainInfoID,
					Height:           blockHeight,
					CovenantBtcPkID:  pkID,
					BTCStakingTxHash: sig.StakingTxHash,
					Timestamp:        blockTimestamp,
				}

				newBcsList = append(newBcsList, newCovenantSignature)
			}

			ch2 <- helper.Result{
				Item:    newBcsList,
				Success: true,
			}
		}(ch1, ch2)
	}

	go func() {
		wg.Wait()
		close(ch1)
		close(ch2)
	}()

	closedCh1 := false
	closedCh2 := false

	for {
		select {
		case msg, ok := <-ch1:
			if !ok {
				closedCh1 = true
				ch1 = nil
			} else {
				btcDelegations := msg.Item.([]model.BabylonBtcDelegation)
				btcDelegationsList = append(btcDelegationsList, btcDelegations...)
			}
		case msg, ok := <-ch2:
			if !ok {
				closedCh2 = true
				ch2 = nil
			} else {
				covenantSigs := msg.Item.([]model.BabylonCovenantSignature)
				covenantSignatureList = append(covenantSignatureList, covenantSigs...)
			}
		}

		// exit loop
		if closedCh1 && closedCh2 {
			fmt.Println("All channels closed. Exiting loop.")
			break
		}
	}

	// 1. Insert Babylon Btc Delegations Tx
	err := idx.btcDelRepo.InsertBabylonBtcDelegationsList(idx.ChainInfoID, btcDelegationsList)
	if err != nil {
		return lastIndexPointerHeight, err
	}

	// 2. update sig status
	err = idx.csRepo.InsertBabylonCovenantSignatureList(idx.ChainInfoID, endHeight, covenantSignatureList)
	if err != nil {
		return lastIndexPointerHeight, err
	}

	idx.updatePrometheusMetrics(covenantSignatureList, btcDelegationsList, endBlockTimestamp)
	idx.Debugf("updated babylon covenant signature in %v block", endBlockTimestamp)
	return endHeight, nil
}
