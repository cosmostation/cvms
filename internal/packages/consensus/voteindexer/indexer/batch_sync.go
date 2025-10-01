package indexer

import (
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/common/function"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/consensus/voteindexer/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (vidx *VoteIndexer) batchSync(lastIndexPointerHeight, newIndexPointerHeight int64) (
	/* new index pointer */ int64,
	/* error */ error,
) {
	if lastIndexPointerHeight >= vidx.Lh.LatestHeight {
		vidx.Debugf("current height is %d and latest height is %d both of them are same, so it'll skip the logic", lastIndexPointerHeight, vidx.Lh.LatestHeight)
		return lastIndexPointerHeight, nil
	}

	// set starntHeight and endHeight for batch sync
	startHeight := newIndexPointerHeight
	endHeight := vidx.Lh.LatestHeight

	// set limit at end-height in this batch sync logic
	if (vidx.Lh.LatestHeight - newIndexPointerHeight) > indexertypes.BatchSyncLimit {
		endHeight = newIndexPointerHeight + indexertypes.BatchSyncLimit
		vidx.Debugf("by batch sync limit, end height will change to %d", endHeight)
	}

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup

	// init summary list
	blockSummaryList := make(map[int64]types.BlockSummary, 0)

	// add -1 new index pointer height data// get current block for collecting last commit signatures
	{
		// this height is last commit height about start height
		height := (startHeight - 1)

		blockHeight, blockTimestamp, blockProposerAddress, _, lastCommitBlockHeight, blockSignatures, err := api.GetBlock(vidx.CommonClient, height)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrap(err, "failed to call api.GetBlock function")
		}

		// get previous tendermint validators for collecting  validators' hex address
		validators, err := api.GetValidators(vidx.CommonClient, height)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrap(err, "failed to call api.GetValidators function")
		}

		blockSummaryList[height] = types.BlockSummary{
			BlockHeight:           blockHeight,
			BlockTimeStamp:        blockTimestamp,
			BlockProposerAddress:  blockProposerAddress,
			LastCommitBlockHeight: lastCommitBlockHeight,
			BlockSignatures:       blockSignatures,
			CosmosValidators:      validators,
		}
	}

	// start to call block data
	for height := startHeight; height <= endHeight; height++ {
		wg.Add(1)
		height := height

		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(vidx.Entry)
			defer wg.Done()

			// get current block for collecting last commit signatures
			blockHeight, blockTimestamp, blockProposerAddress, _, lastCommitBlockHeight, blockSignatures, err := api.GetBlock(vidx.CommonClient, height)
			if err != nil {
				vidx.Errorf("failed to call at %d height data, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// get previous tendermint validators for collecting  validators' hex address
			validators, err := api.GetValidators(vidx.CommonClient, height)
			if err != nil {
				vidx.Errorf("failed to call at %d height data, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			ch <- helper.Result{
				Item: types.BlockSummary{
					BlockHeight:           blockHeight,
					BlockTimeStamp:        blockTimestamp,
					BlockProposerAddress:  blockProposerAddress,
					LastCommitBlockHeight: lastCommitBlockHeight,
					BlockSignatures:       blockSignatures,
					CosmosValidators:      validators,
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

	// collect block summary data into block summary list
	errorCount := 0
	for r := range ch {
		if r.Success {
			item := r.Item.(types.BlockSummary)
			blockSummaryList[item.BlockHeight] = item
			continue
		}
		errorCount++
	}

	// check error count
	if errorCount > 0 {
		return lastIndexPointerHeight, errors.Errorf("failed to collect batch block data, total errors: %d", errorCount)
	}

	// if there are new hex address in current block, collect their validator hex address to save in database
	isNewValidator := false
	newValidatorAddressMap := make(map[string]bool)
	for _, blockInfo := range blockSummaryList {
		// check proposer address
		_, exist := vidx.Vim[blockInfo.BlockProposerAddress]
		if !exist {
			// vidx.Debugf("the proposer %s address isn't in current validator info table, it will be gonna added into the table", blockInfo.BlockProposerAddress)
			newValidatorAddressMap[blockInfo.BlockProposerAddress] = true
			isNewValidator = true
		}
		// check validators address
		for _, validator := range blockInfo.CosmosValidators {
			_, exist := vidx.Vim[validator.Address]
			if !exist {
				vidx.Debugf("the miss validator %s/%s address isn't in current validator info table, it will be gonna added into the table", validator.Address, validator.Pubkey)
				newValidatorAddressMap[validator.Address] = true
				isNewValidator = true
			}
		}
	}

	// this logic will be progressed only when there are new tendermint validators in this block
	if isNewValidator {
		newValidatorInfoList, err := function.MakeValidatorInfoList(vidx.CommonApp,
			vidx.ChainID, vidx.ChainInfoID,
			vidx.ChainName, vidx.IsConsumer,
			newValidatorAddressMap)
		if err != nil {
			errors.Wrap(err, "failed to make validator info list")
		}
		vidx.Debugf("insert new tendermint validators: %d", len(newValidatorInfoList))
		// insert new validators' proposer address into the validator info table
		err = vidx.repo.InsertValidatorInfoList(newValidatorInfoList)
		if err != nil {
			// NOTE: fetch again validator_info list, actually already inserted the list by other indexer service
			vidx.FetchValidatorInfoList()
			return lastIndexPointerHeight, errors.WithStack(err)
		}

		// get already saved tendermint validator list for mapping validators ids
		validatorInfoList, err := vidx.repo.GetValidatorInfoListByChainInfoID(vidx.ChainInfoID)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrap(err, "failed to get new validator info list after inserting new hex address list")
		}

		for _, validator := range validatorInfoList {
			vidx.Vim[validator.HexAddress] = int64(validator.ID)
		}

		vidx.Debugf("changed vim length: %d", len(vidx.Vim))
	}

	ValidatorVoteList := make([]model.ValidatorVote, 0)
	for height := startHeight; height <= endHeight; height++ {
		lastCommitHeight := (height - 1)

		// validate check
		if blockSummaryList[height].LastCommitBlockHeight != blockSummaryList[lastCommitHeight].BlockHeight {
			vidx.Panicln("unexpected errors: not matched block data in block summary list")
		}

		tempValidatorVoteList, err := makeValidatorVoteList(
			// logger
			vidx.Entry,
			// vms instance data
			vidx.ChainInfoID,
			vidx.Vim,
			// previous block data
			blockSummaryList[lastCommitHeight].BlockHeight,
			blockSummaryList[lastCommitHeight].BlockTimeStamp,
			blockSummaryList[lastCommitHeight].BlockProposerAddress,
			blockSummaryList[lastCommitHeight].CosmosValidators,
			// current block data
			blockSummaryList[height].BlockSignatures,
		)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrapf(err, "failed to make temp validator miss list at %d height", height)
		}

		ValidatorVoteList = append(ValidatorVoteList, tempValidatorVoteList...)
	}

	//  only loggic when there are miss validators in the network
	if len(ValidatorVoteList) > 0 {
		vidx.Infof("found %d miss validators from %d to %d in the network", len(ValidatorVoteList), startHeight, endHeight)
	}

	// NOTE: if solo validator mode, we don't need to insert all validotors vote status.
	// so, filter statues by moniker
	if len(vidx.Monikers) > 0 {
		vidx.Infof("Voteindex Moniker Length: %d, %f", len(vidx.Monikers), vidx.Monikers)
		// if not init monikerIDMap
		if len(vidx.MonikerIDMap) != len(vidx.Monikers) {
			// init monikerIDMap
			validatorInfoList, err := vidx.repo.GetValidatorInfoListByMonikers(vidx.ChainInfoID, vidx.Monikers)
			if err != nil {
				vidx.Info("Voteindex Moniker error by DB")
				return lastIndexPointerHeight, errors.Wrap(err, "failed to get validator_info list by monikers")
			}
			monikerIDMap := make(indexertypes.MonikerIDMap)
			for _, vi := range validatorInfoList {
				monikerIDMap[vi.ID] = true
			}
			vidx.Infof("Voteindex Moniker: %f", validatorInfoList)
			// restore monikerIDMap in voteindexer struct, for reusing
			vidx.MonikerIDMap = monikerIDMap
		}

		// override for solo validator
		ValidatorVoteList = filterValidatorVoteListByMonikers(vidx.MonikerIDMap, ValidatorVoteList)
		vidx.Infof("Override Voteindex Moniker: %f", ValidatorVoteList)
	} else {
		vidx.Info("ALL MODE!!")
	}

	// need to save list and new pointer
	err := vidx.repo.InsertValidatorVoteList(vidx.ChainInfoID, blockSummaryList[endHeight].BlockHeight, ValidatorVoteList)
	if err != nil {
		return lastIndexPointerHeight, errors.Wrapf(err, "failed to insert from %d to %d height", startHeight, endHeight)
	}

	// update metrics
	vidx.updateRootMetrics(blockSummaryList[endHeight].BlockHeight, blockSummaryList[endHeight].BlockTimeStamp)
	return blockSummaryList[endHeight].BlockHeight, nil
}

// make validator miss list with current & previous block data
// return list is will be inserted in the database
func makeValidatorVoteList(
	// vmc instance data
	vml *logrus.Entry,
	chainInfoID int64,
	validatorIDMap indexertypes.ValidatorIDMap,
	// previous block data
	lastCommitBlockHeight int64,
	lastCommitBlockTimestamp time.Time,
	lastCommitBlockProposerAddress string,
	lastCommitValidators []types.CosmosValidator,
	// current block data
	blockSignatures []types.Signature,
) ([]model.ValidatorVote, error) {
	ValidatorVoteList := make([]model.ValidatorVote, 0)
	lastCommitBlockProposerAddressID, exist := validatorIDMap[lastCommitBlockProposerAddress]
	if !exist {
		return nil, errors.New("failed to find block proposer hex address id in validator id maps")
	}

	for idx, validator := range lastCommitValidators {
		// Note that it used to be block.Block.LastCommit.Precommits[i] == nil
		if blockSignatures[idx].Signature == nil {
			vml.Debugf(
				`found miss validator <idx: %d, address: %s> in this block <height: %d>`,
				validatorIDMap[validator.Address], validator.Address, lastCommitBlockHeight,
			)

			validatorHexAddressID, exist := validatorIDMap[validator.Address]
			if !exist {
				return nil, errors.New("failed to find missed validators hex address id in validator id maps")
			}

			ValidatorVoteList = append(ValidatorVoteList, model.ValidatorVote{
				ChainInfoID: chainInfoID,
				// current block data
				ValidatorHexAddressID: validatorHexAddressID,
				// previous block data
				Height:    lastCommitBlockHeight,
				Timestamp: lastCommitBlockTimestamp,
				Status:    model.Missed,
			})
		} else {
			validatorHexAddressID, exist := validatorIDMap[validator.Address]
			if !exist {
				return nil, errors.New("failed to find missed validators hex address id in validator id maps")
			}

			if lastCommitBlockProposerAddressID == validatorHexAddressID {
				// for only proposer
				ValidatorVoteList = append(ValidatorVoteList, model.ValidatorVote{
					ChainInfoID: chainInfoID,
					// current block data
					ValidatorHexAddressID: validatorHexAddressID,
					// previous block data
					Height:    lastCommitBlockHeight,
					Timestamp: lastCommitBlockTimestamp,
					Status:    model.Proposed,
				})
			} else {
				// for voters, not proposer
				ValidatorVoteList = append(ValidatorVoteList, model.ValidatorVote{
					ChainInfoID: chainInfoID,
					// current block data
					ValidatorHexAddressID: validatorHexAddressID,
					// previous block data
					Height:    lastCommitBlockHeight,
					Timestamp: lastCommitBlockTimestamp,
					Status:    model.Voted,
				})
			}
		}
	}

	return ValidatorVoteList, nil
}

func filterValidatorVoteListByMonikers(monikerIDMap indexertypes.MonikerIDMap, vvList []model.ValidatorVote) []model.ValidatorVote {
	// already inited monikerIDMap just filter validator vote by moniker id maps
	newValidatorVoteList := make([]model.ValidatorVote, 0)
	for _, vv := range vvList {
		// // only append validaor vote in package monikers
		_, exist := monikerIDMap[vv.ValidatorHexAddressID]
		if exist {
			newValidatorVoteList = append(newValidatorVoteList, vv)
		}
	}
	return newValidatorVoteList
}
