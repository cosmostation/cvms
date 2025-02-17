package indexer

import (
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common/api"
	indexermodel "github.com/cosmostation/cvms/internal/common/indexer/model"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/helper"

	"github.com/cosmostation/cvms/internal/packages/babylon-btc-lightclient/model"
	"github.com/pkg/errors"
)

func (idx *BTCLightClientIndexer) batchSync(lastIndexPoint int64) (
	/* new index pointer */ int64,
	/* error */ error,
) {
	if lastIndexPoint >= idx.Lh.LatestHeight {
		idx.Infof("current height is %d and latest height is %d both of them are same, so it'll skip the logic", lastIndexPoint, idx.Lh.LatestHeight)
		return lastIndexPoint, nil
	}

	// set starntHeight and endHeight for batch sync
	startHeight := (lastIndexPoint + 1)
	endHeight := idx.Lh.LatestHeight

	// set limit at end-height in this batch sync logic
	if (idx.Lh.LatestHeight - startHeight) > indexertypes.BatchSyncLimit {
		endHeight = startHeight + indexertypes.BatchSyncLimit
		idx.Debugf("by batch sync limit, end height will change to %d", endHeight)
	}

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	wg := sync.WaitGroup{}
	bieSummaryList := make(map[int64]BTCInsertEventSummary, 0)

	// start to call block results
	for h := startHeight; h <= endHeight; h++ {
		wg.Add(1)
		height := h

		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(idx.Entry)
			defer wg.Done()

			txsEvents, _, err := api.GetBlockResults(idx.CommonClient, height)
			if err != nil {
				idx.Errorf("failed to call at %d height data, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// NOTE: bieList means btc insert events list
			bieList, err := filterBTCLightClientEvents(txsEvents)
			if err != nil {
				idx.Errorf("failed to call at %d height data, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// if there is no btc insert event in the block, just skip this height to index
			if len(bieList) == 0 {
				ch <- helper.Result{
					Item:    BTCInsertEventSummary{true, height, bieList},
					Success: true,
				}
			}

			ch <- helper.Result{
				Item:    BTCInsertEventSummary{false, height, bieList},
				Success: true,
			}
		}(ch)

		time.Sleep(10 * time.Millisecond)
	}

	// close channel
	go func() {
		wg.Wait()
		close(ch)
	}()

	// collect block summary data into block summary list
	errorCount := 0
	for r := range ch {
		if r.Success {
			item := r.Item.(BTCInsertEventSummary)
			bieSummaryList[item.BlockHeight] = item
			continue
		}
		errorCount++
	}

	// check error count
	if errorCount > 0 {
		return lastIndexPoint, errors.Errorf("failed to collect batch block data, total errors: %d", errorCount)
	}

	// if there are new hex address in current block, collect their validator hex address to save in database
	isNewReporter := false
	newRepoterInfoList := make([]indexermodel.ValidatorInfo, 0)
	for _, bie := range bieSummaryList {
		if bie.skip {
			continue
		}

		// check reporter address
		for _, e := range bie.BTCInsertEvents {
			_, exist := idx.Vim[e.ReporterAddress]
			if !exist {
				idx.Debugf("the reporter %s address isn't in current validator info table, the address will be added into the meta table", e.ReporterAddress)
				newRepoterInfoList = append(newRepoterInfoList, indexermodel.ValidatorInfo{
					ChainInfoID:     idx.ChainInfoID,
					HexAddress:      e.ReporterAddress[:40],
					OperatorAddress: e.ReporterAddress,
					Moniker:         "Babylon Vigilante Repoter",
				})
				isNewReporter = true
			}
		}
	}

	// this logic will be progressed only when there are new tendermint validators in this block
	if isNewReporter {
		idx.Debugf("insert new vigilante reporters: %d", len(newRepoterInfoList))
		err := idx.InsertValidatorInfoList(newRepoterInfoList)
		if err != nil {
			// NOTE: fetch again validator_info list, actually already inserted the list by other indexer service
			idx.FetchValidatorInfoList()
			return lastIndexPoint, errors.Wrap(err, "failed to insert new reporter list")
		}

		// get already saved tendermint validator list for mapping validators ids
		repoterInfoList, err := idx.GetValidatorInfoListByChainInfoID(idx.ChainInfoID)
		if err != nil {
			return lastIndexPoint, errors.Wrap(err, "failed to get new reporter info list after inserting new hex address list")

		}

		for _, reporter := range repoterInfoList {
			idx.Vim[reporter.OperatorAddress] = int64(reporter.ID)
		}

		idx.Debugf("changed vim length: %d", len(idx.Vim))
	}

	modelList := make([]model.BabylonBTCRoll, 0)
	for _, bie := range bieSummaryList {
		for _, e := range bie.BTCInsertEvents {
			reporterID, exist := idx.Vim[e.ReporterAddress]
			if !exist {
				return lastIndexPoint, errors.New("failed to find reporter ID in indexer ID maps")
			}

			modelList = append(modelList, model.BabylonBTCRoll{
				ChainInfoID: idx.ChainInfoID,
				Height:      bie.BlockHeight,
				ReporterID:  reporterID,
				HeaderCount: len(e.BTCHeaders),
				BTCHeaders:  e.ToHeadersStringSlice(),
			})

		}
	}

	//  only loggic when there are miss validators in the network
	if len(modelList) > 0 {
		idx.Infof("found %d BTC insert events from %d to %d in the network", len(modelList), startHeight, endHeight)
	}

	// insert model list and update index pointer
	err := idx.InsertBabylonBTCRollList(idx.ChainInfoID, endHeight, modelList)
	if err != nil {
		return lastIndexPoint, errors.Wrapf(err, "failed to insert from %d to %d height", startHeight, endHeight)
	}

	idx.updatePrometheusMetrics(endHeight)
	return endHeight, nil
}
