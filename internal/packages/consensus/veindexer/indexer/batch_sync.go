package indexer

import (
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/common/function"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/helper"
	sdkhelper "github.com/cosmostation/cvms/internal/helper/sdk"

	"github.com/cosmostation/cvms/internal/packages/consensus/veindexer/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NOTE: N-1 VE(vote extension) are consumed to build proposal in n block.
// so that, success VE voting commit in specific block would be already decided in previous block.
// ref; https://docs.skip.build/connect/learn/architecture
func (veidx *VEIndexer) batchSync(lastIndexPointerHeight, newIndexPointerHeight int64) (
	/* new index pointer */ int64,
	/* error */ error,
) {
	if lastIndexPointerHeight >= veidx.Lh.LatestHeight {
		veidx.Debugf("current height is %d and latest height is %d both of them are same, so it'll skip the logic", lastIndexPointerHeight, veidx.Lh.LatestHeight)
		return lastIndexPointerHeight, nil
	}

	// set starntHeight and endHeight for batch sync
	startHeight := newIndexPointerHeight
	endHeight := veidx.Lh.LatestHeight

	// set limit at end-height in this batch sync logic
	if (veidx.Lh.LatestHeight - newIndexPointerHeight) > indexertypes.BatchSyncLimit {
		endHeight = newIndexPointerHeight + indexertypes.BatchSyncLimit
		veidx.Debugf("by batch sync limit, end height will change to %d", endHeight)
	}

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	var wg sync.WaitGroup

	// init summary list
	blockSummaryList := make(map[int64]types.BlockSummary, 0)

	// add -1 new index pointer height data, get current block for collecting last commit signatures
	{
		// this height is last commit height about start height
		height := (startHeight - 1)

		blockHeight, blockTimestamp, blockProposerAddress, blockTxs, lastCommitBlockHeight, blockSignatures, err := api.GetBlock(veidx.CommonClient, height)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrap(err, "failed to call api.GetBlock function")
		}

		// get previous tendermint validators for collecting  validators' hex address
		validators, err := api.GetValidators(veidx.CommonClient, height)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrap(err, "failed to call api.GetValidators function")
		}

		blockSummaryList[height] = types.BlockSummary{
			BlockHeight:           blockHeight,
			BlockTimeStamp:        blockTimestamp,
			BlockProposerAddress:  blockProposerAddress,
			Txs:                   blockTxs,
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
			defer helper.HandleOutOfNilResponse(veidx.Entry)
			defer wg.Done()

			// get current block for collecting last commit signatures
			blockHeight, blockTimestamp, blockProposerAddress, blockTxs, lastCommitBlockHeight, blockSignatures, err := api.GetBlock(veidx.CommonClient, height)
			if err != nil {
				veidx.Errorf("failed to call at %d height data, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// get previous tendermint validators for collecting  validators' hex address
			validators, err := api.GetValidators(veidx.CommonClient, height)
			if err != nil {
				veidx.Errorf("failed to call at %d height data, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			ch <- helper.Result{
				Item: types.BlockSummary{
					BlockHeight:           blockHeight,
					BlockTimeStamp:        blockTimestamp,
					BlockProposerAddress:  blockProposerAddress,
					Txs:                   blockTxs,
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
		_, exist := veidx.Vim[blockInfo.BlockProposerAddress]
		if !exist {
			// veidx.Debugf("the proposer %s address isn't in current validator info table, it will be gonna added into the table", blockInfo.BlockProposerAddress)
			newValidatorAddressMap[blockInfo.BlockProposerAddress] = true
			isNewValidator = true
		}

		// check validators address
		for _, validator := range blockInfo.CosmosValidators {
			_, exist := veidx.Vim[validator.Address]
			if !exist {
				veidx.Debugf("the miss validator %s/%s address isn't in current validator info table, it will be gonna added into the table", validator.Address, validator.Pubkey)
				newValidatorAddressMap[validator.Address] = true
				isNewValidator = true
			}
		}
	}

	// this logic will be progressed only when there are new tendermint validators in this block
	if isNewValidator {
		newValidatorInfoList, err := function.MakeValidatorInfoList(veidx.CommonApp,
			veidx.ChainID, veidx.ChainInfoID,
			veidx.ChainName, veidx.IsConsumer,
			newValidatorAddressMap)
		if err != nil {
			errors.Wrap(err, "failed to make validator info list")
		}

		// insert new validators' proposer address into the validator info table
		err = veidx.repo.InsertValidatorInfoList(newValidatorInfoList)
		if err != nil {
			// NOTE: fetch again validator_info list, actually already inserted the list by other indexer service
			veidx.FetchValidatorInfoList()
			return lastIndexPointerHeight, errors.Wrap(err, "failed to insert new hex address list")
		}

		// get already saved tendermint validator list for mapping validators ids
		validatorInfoList, err := veidx.repo.GetValidatorInfoListByChainInfoID(veidx.ChainInfoID)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrap(err, "failed to get new validator info list after inserting new hex address list")
		}

		for _, validator := range validatorInfoList {
			veidx.Vim[validator.HexAddress] = int64(validator.ID)
		}

		veidx.Infof("changed vim length: %d", len(veidx.Vim))
	}

	vevList := make([]model.ValidatorExtensionVote, 0)
	for height := startHeight; height <= endHeight; height++ {
		lastCommitHeight := (height - 1)

		// NOTE: validate check
		if blockSummaryList[height].LastCommitBlockHeight != blockSummaryList[lastCommitHeight].BlockHeight {
			veidx.Panicln("unexpected errors: not matched block data in block summary list")
		}

		// NOTE: validation check2
		// chains which be enabled VE should have one tx in the block at least
		if len(blockSummaryList[height].Txs) < 1 {
			return lastIndexPointerHeight, errors.Errorf("VE should be a first tx, but there is no txs in the block at %d height", height)
		}

		tempVEVList, err := makeValidatorExtensionList(
			// logger
			veidx.Entry,
			// vms instance data
			veidx.ChainInfoID,
			veidx.Vim,
			// previous block data
			blockSummaryList[lastCommitHeight].BlockHeight,
			blockSummaryList[lastCommitHeight].BlockTimeStamp,
			// current block data
			blockSummaryList[height].Txs,
		)
		if err != nil {
			return lastIndexPointerHeight, errors.Wrapf(err, "failed to make temp validator extension list at %d height", height)
		}

		vevList = append(vevList, tempVEVList...)
	}

	//  only loggic when there are miss validators in the network
	if len(vevList) > 0 {
		veidx.Infof("found %d miss validators from %d to %d in the network", len(vevList), startHeight, endHeight)
	}

	// NOTE: if solo validator mode, we don't need to insert all validotors vote status.
	// so, filter statues by moniker
	if len(veidx.Monikers) > 0 {
		// if not init monikerIDMap
		if len(veidx.MonikerIDMap) != len(veidx.Monikers) {
			// init monikerIDMap
			validatorInfoList, err := veidx.repo.GetValidatorInfoListByMonikers(veidx.ChainInfoID, veidx.Monikers)
			if err != nil {
				return lastIndexPointerHeight, errors.Wrap(err, "failed to get validator_info list by monikers")
			}
			monikerIDMap := make(indexertypes.MonikerIDMap)
			for _, vi := range validatorInfoList {
				monikerIDMap[vi.ID] = true
			}

			// restore monikerIDMap in voteindexer struct, for reusing
			veidx.MonikerIDMap = monikerIDMap
		}

		// override for solo validator
		vevList = filterValidatorVoteListByMonikers(veidx.MonikerIDMap, vevList)
	}

	// need to save list and new pointer
	err := veidx.repo.InsertValidatorExtensionVoteList(veidx.ChainInfoID, blockSummaryList[endHeight].BlockHeight, vevList)
	if err != nil {
		return lastIndexPointerHeight, errors.Wrapf(err, "failed to insert from %d to %d height", startHeight, endHeight)
	}

	// update metrics
	veidx.updateRootMetrics(blockSummaryList[endHeight].BlockHeight, blockSummaryList[endHeight].BlockTimeStamp)
	return blockSummaryList[endHeight].BlockHeight, nil
}

// make validator_extension list with current & previous block data
// return list is will be inserted in the database
func makeValidatorExtensionList(
	// vmc instance data
	vml *logrus.Entry,
	chainInfoID int64,
	validatorIDMap indexertypes.ValidatorIDMap,
	// previous block data
	lastCommitBlockHeight int64,
	lastCommitBlockTimestamp time.Time,
	// current block txs data
	blockTxs []types.Tx,
) ([]model.ValidatorExtensionVote, error) {
	// after checking len(txs) >= 1
	// firstly, we should decode first tx(txs[0]) for getting VE statuses
	voteExtensionTx := blockTxs[0]
	ves, err := sdkhelper.DecodingVoteExtensionTx(string(voteExtensionTx))
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode vote extension from txs[0]")
	}

	vevList := make([]model.ValidatorExtensionVote, 0)
	for _, ve := range ves {
		if ve.Signature == nil {
			vml.Debugf(
				`found miss validator <idx: %d, address: %s> in this block <height: %d>`,
				validatorIDMap[ve.Address], ve.Address, lastCommitBlockHeight,
			)

			validatorHexAddressID, exist := validatorIDMap[ve.Address]
			if !exist {
				vml.Debugf("debug: %v", ve)
				return nil, errors.New("failed to find missed validators hex address id in validator id maps")
			}

			vevList = append(vevList, model.ValidatorExtensionVote{
				ChainInfoID:           chainInfoID,
				ValidatorHexAddressID: validatorHexAddressID,
				Height:                lastCommitBlockHeight,
				Timestamp:             lastCommitBlockTimestamp,
				Status:                ve.BlockCommitFlag,
				VELength:              len(ve.VoteExtension),
			})
		} else {
			validatorHexAddressID, exist := validatorIDMap[ve.Address]
			if !exist {
				vml.Debugf("debug: %v", ve)
				return nil, errors.New("failed to find missed validators hex address id in validator id maps")
			}

			// for committed voters
			vevList = append(vevList, model.ValidatorExtensionVote{
				ChainInfoID:           chainInfoID,
				ValidatorHexAddressID: validatorHexAddressID,
				Height:                lastCommitBlockHeight,
				Timestamp:             lastCommitBlockTimestamp,
				Status:                ve.BlockCommitFlag,
				VELength:              len(ve.VoteExtension),
			})
		}
	}

	return vevList, nil
}

func filterValidatorVoteListByMonikers(monikerIDMap indexertypes.MonikerIDMap, vvList []model.ValidatorExtensionVote) []model.ValidatorExtensionVote {
	// already inited monikerIDMap just filter validator vote by moniker id maps
	newValidatorVoteList := make([]model.ValidatorExtensionVote, 0)
	for _, vv := range vvList {
		// // only append validaor vote in package monikers
		_, exist := monikerIDMap[vv.ValidatorHexAddressID]
		if exist {
			newValidatorVoteList = append(newValidatorVoteList, vv)
		}
	}
	return newValidatorVoteList
}
