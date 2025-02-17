package indexer

import (
	"sync"
	"time"

	commonapi "github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/common/function"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/duty/finality-provider-indexer/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const finalitySigTimeout = 3

// NOTE: in this package, the missed height means fp didn't broadcast the finality sig tx in height+1 block
// for example, I didn't broadcast the tx at 100 block, that will make missed block for 99 height.
func (idx *FinalityProviderIndexer) batchSync(lastIndexPointerHeight int64) (
	/* new index pointer */ int64,
	/* error */ error,
) {
	// set starntHeight and endHeight for batch sync
	// NOTE: end height will use latest height -1 for waiting latest vote status
	startHeight := (lastIndexPointerHeight + 1)
	endHeight := (idx.Lh.LatestHeight - finalitySigTimeout)
	if startHeight > endHeight {
		idx.Infof("no need to sync from %d height to %d height, so it'll skip the logic", startHeight, endHeight)
		return lastIndexPointerHeight, nil
	}

	// set limit at end-height in this batch sync logic
	if (endHeight - startHeight) > indexertypes.BatchSyncLimit {
		endHeight = startHeight + indexertypes.BatchSyncLimit
		idx.Infof("by batch sync limit, end height will change to %d", endHeight)
	}

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup

	// init summary list
	FinalityVoteSummaryList := make(map[int64]FinalityVoteSummary, 0)

	// start to call block data
	for height := startHeight; height <= endHeight; height++ {
		wg.Add(1)
		missedHeight := height

		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(idx.Entry)
			defer wg.Done()

			// get current block for collecting last commit signatures
			fps, err := commonapi.GetActiveFinalityProviderByHeight(idx.CommonClient, missedHeight)
			if err != nil {
				idx.Errorf("failed to call at %d height data, %s", missedHeight, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// make a map by fp votes with default value(false)
			fpVoteMap := make(fpVoteMap, len(fps))
			for _, fp := range fps {
				fpVoteMap[fp.BtcPkHex] = 0 // missed
			}

			// get previous tendermint validators for collecting  validators' hex address
			btcPKs, err := commonapi.GetFinalityVotesByHeight(idx.CommonClient, missedHeight)
			if err != nil {
				idx.Errorf("failed to call at %d height data, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// if empty response, the height is not updated yet wait for while
			if len(btcPKs) == 0 {
				idx.Warnf("finality provider vote status was not updated yet for %d height", missedHeight)
				ch <- helper.Result{Item: nil, Success: false}
			}

			// if the pk is existed in the votings, update the value for fp
			for _, pk := range btcPKs {
				fpVoteMap[pk] = 1 // voted
			}

			idx.Infof("in %d+1 block, total fp: %d but voted only %d", missedHeight, len(fps), len(btcPKs))

			ch <- helper.Result{
				Item: FinalityVoteSummary{
					BlockHeight:           height,
					FinalityProviderVotes: fpVoteMap,
				},
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

	// collect summary data into summary list
	errorCount := 0
	for r := range ch {
		if r.Success {
			item := r.Item.(FinalityVoteSummary)
			FinalityVoteSummaryList[item.BlockHeight] = item
			continue
		}
		errorCount++
	}

	// check error count
	if errorCount > 0 {
		return lastIndexPointerHeight, errors.Errorf("failed to collect batch block data, total errors: %d", errorCount)
	}

	// if there are new btc pk in current block, collect their validator hex address to save in database
	isNewFinalityProvider := false
	newFinalityProviderMap := make(map[string]bool, 0)
	for _, item := range FinalityVoteSummaryList {
		for pk := range item.FinalityProviderVotes {
			_, exist := idx.Vim[pk]
			if !exist {
				idx.Debugf("the miss finality provider %s isn't in current validator info table, it will be gonna added into the table", pk)
				newFinalityProviderMap[pk] = true
				isNewFinalityProvider = true
			}
		}
	}

	// this logic will be progressed only when there are new tendermint validators in this block
	if isNewFinalityProvider {
		newfpInfoList, err := function.MakeFinalityProviderInfoList(idx.CommonClient, idx.ChainInfoID, newFinalityProviderMap)
		if err != nil {
			errors.Wrap(err, "failed to make validator info list")
		}

		idx.Debugf("insert new finality providers: %d", len(newfpInfoList))

		// insert new validators' proposer address into the validator info table
		err = idx.repo.InsertFinalityProviderInfoList(newfpInfoList)
		if err != nil {
			// NOTE: fetch again validator_info list, actually already inserted the list by other indexer service
			idx.FetchValidatorInfoList()
			return lastIndexPointerHeight, errors.WithStack(err)
		}

		// get already saved tendermint validator list for mapping validators ids
		fpInfoList, err := idx.repo.GetFinalityProviderInfoListByChainInfoID(idx.ChainInfoID)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrap(err, "failed to get new validator info list after inserting new hex address list")

		}

		for _, fp := range fpInfoList {
			idx.Vim[fp.BTCPKs] = int64(fp.ID)
		}

		idx.Debugf("changed vim length: %d", len(idx.Vim))
	}

	fpvList := make([]model.BabylonFinalityProviderVote, 0)
	for height := startHeight; height <= endHeight; height++ {
		if height != FinalityVoteSummaryList[height].BlockHeight {
			idx.Panicln("unexpected")
		}

		tempValidatorVoteList, err := makeBabylonFinalityProviderVoteList(
			idx.Entry,
			idx.ChainInfoID,
			idx.Vim,
			FinalityVoteSummaryList[height].BlockHeight,
			FinalityVoteSummaryList[height].FinalityProviderVotes,
		)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrapf(err, "failed to make temp validator miss list at %d height", height)
		}

		fpvList = append(fpvList, tempValidatorVoteList...)
	}

	// NOTE: if solo validator mode, we don't need to insert all validotors vote status.
	// so, filter statues by moniker
	if len(idx.Monikers) > 0 {
		// if not init monikerIDMap
		if len(idx.MonikerIDMap) != len(idx.Monikers) {
			// init monikerIDMap
			validatorInfoList, err := idx.repo.GetFinalityProviderInfoListByMonikers(idx.ChainInfoID, idx.Monikers)
			if err != nil {
				return lastIndexPointerHeight, errors.Wrap(err, "failed to get validator_info list by monikers")
			}
			monikerIDMap := make(indexertypes.MonikerIDMap)
			for _, vi := range validatorInfoList {
				monikerIDMap[vi.ID] = true
			}
			// restore monikerIDMap in voteindexer struct, for reusing
			idx.MonikerIDMap = monikerIDMap
		}

		// override for solo validator
		fpvList = filterValidatorVoteListByMonikers(idx.MonikerIDMap, fpvList)
	}

	// need to save list and new pointer
	err := idx.repo.InsertFinalityProviderVoteList(idx.ChainInfoID, FinalityVoteSummaryList[endHeight].BlockHeight, fpvList)
	if err != nil {
		return lastIndexPointerHeight, errors.Wrapf(err, "failed to insert from %d to %d height", startHeight, endHeight)
	}

	// update metrics
	// idx.updatePrometheusMetrics(blockSummaryList[endHeight].BlockHeight, blockSummaryList[endHeight].BlockTimeStamp)
	return FinalityVoteSummaryList[endHeight].BlockHeight, nil
}

func makeBabylonFinalityProviderVoteList(
	l *logrus.Entry,
	chainInfoID int64,
	validatorIDMap indexertypes.ValidatorIDMap,
	missedBlockHeight int64,
	fpVotes fpVoteMap,
) ([]model.BabylonFinalityProviderVote, error) {
	bfpVoteList := make([]model.BabylonFinalityProviderVote, 0)
	for btcPK, status := range fpVotes {
		fpPKID, exist := validatorIDMap[btcPK]
		if !exist {
			return nil, errors.Errorf("failed %s's id in validatorIDMap", btcPK)
		}
		if status == 0 {
			l.Debugf("found missed finality provider idx: %d, address: %s in this block height: %d", validatorIDMap[btcPK], btcPK, missedBlockHeight)
		}
		bfpVoteList = append(bfpVoteList, model.BabylonFinalityProviderVote{
			ChainInfoID:          chainInfoID,
			Height:               missedBlockHeight,
			FinalityProviderPKID: fpPKID,
			Status:               int64(status),
			CreatedTime:          time.Now(),
		})
	}
	return bfpVoteList, nil
}

func filterValidatorVoteListByMonikers(monikerIDMap indexertypes.MonikerIDMap, bfpvList []model.BabylonFinalityProviderVote) []model.BabylonFinalityProviderVote {
	// already inited monikerIDMap just filter validator vote by moniker id maps
	newFinalityProviderVoteList := make([]model.BabylonFinalityProviderVote, 0)
	for _, bfpv := range bfpvList {
		// // only append validaor vote in package monikers
		_, exist := monikerIDMap[bfpv.FinalityProviderPKID]
		if exist {
			newFinalityProviderVoteList = append(newFinalityProviderVoteList, bfpv)
		}
	}
	return newFinalityProviderVoteList
}
