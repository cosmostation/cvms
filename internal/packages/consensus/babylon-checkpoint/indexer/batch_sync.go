package indexer

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	tmtypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/common/function"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-checkpoint/model"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

const InitEpochInterval = 50

// NOTE: babylon checkpoint will be created at epoch + 1 block in each epoch
// first we can get the epoch parameter from the chain
// 1. get current epoch /babylon/epoching/v1/epochs/current_epoch -> 22
// 2. get epoch  /babylon/epoching/v1/epochs/{epoch_num}
// and then, query the first decoded tx in the block
// 2. https://lcd-office.cosmostation.io/babylon-testnet/cosmos/tx/v1beta1/txs/block/721?pagination.limit=1
// 3. check BLOCK_ID_FLAG_COMMIT flag with address
// 4. make a list for checkpoint
// N-1 VE(vote extension) are consumed to build proposal in n block.
// so that, success VE voting commit in specific block would be already decided in previous block.
// ref; https://docs.skip.build/connect/learn/architecture
func (idx *CheckpointIndexer) batchSync(lastIndexPointerEpoch int64) (
	/* new index pointer */ int64,
	/* error */ error,
) {
	newIndexerPointerEpoch := lastIndexPointerEpoch + 1
	ctx := context.Background()
	resp, err := idx.APIClient.Get(ctx, CurrentEpochQueryPath)
	if err != nil {
		return lastIndexPointerEpoch, errors.Wrap(err, "failed to call current epoch api")
	}

	currentEpoch, _, err := ParseCurrentEpoch(resp)
	if err != nil {
		return lastIndexPointerEpoch, errors.Wrap(err, "failed to parse current epoch response")
	}

	if (currentEpoch - newIndexerPointerEpoch) > InitEpochInterval {
		newIndexerPointerEpoch = currentEpoch - InitEpochInterval
		idx.Infof("changed index pointer epoch: %d to ignore old epoch data", newIndexerPointerEpoch)
	}

	lastFinalizedEpoch := (currentEpoch - 1)
	idx.Debugf("current epoch: %d and last finalized epoch: %d", currentEpoch, lastFinalizedEpoch)

	// NOTE: if (current epoch -1) > lastEpoch -> the indexer needs to sync more epoch
	if newIndexerPointerEpoch > lastFinalizedEpoch {
		idx.Infof(
			"last finalized epoch is %d and last db epoch is %d. nothing to sync epochs, so it'll skip the logic",
			lastFinalizedEpoch, lastIndexPointerEpoch,
		)
		sleepDuration := (time.Minute * 10)
		idx.Infof("sleep %s sec...", sleepDuration.String())
		time.Sleep(sleepDuration)
		return lastIndexPointerEpoch, nil
	}

	idx.Infof("last finalized epoch(current_epoch -1) is %d and last index pointer epoch is %d", lastFinalizedEpoch, lastIndexPointerEpoch)
	for epoch := int64(0); epoch <= lastFinalizedEpoch; epoch++ {
		if epoch <= 1 {
			continue
		}

		if newIndexerPointerEpoch > epoch {
			idx.Debugf("skip %d epoch. the epoch was already stored in the DB", epoch)
			continue
		}

		idx.Debugf("sync epoch: %d", epoch)
		resp, err := idx.APIClient.Get(ctx, EpochQueryPath(epoch))
		if err != nil {
			return lastIndexPointerEpoch, errors.Wrap(err, "failed to get epoch data")
		}

		firstBlockHeightInEpoch, _, err := ParseEpoch(resp)
		if err != nil {
			return lastIndexPointerEpoch, errors.Wrap(err, "failed to get epoch data")
		}

		prevBlockHeight, _, preBlockProposerAddress, _, _, _, err := api.GetBlock(idx.CommonClient, firstBlockHeightInEpoch-1)
		if err != nil {
			idx.Errorf("failed to call at %d height data, %s", prevBlockHeight, err)
			return lastIndexPointerEpoch, err
		}

		// get previous tendermint validators for collecting  validators' hex address
		validators, err := api.GetValidators(idx.CommonClient, prevBlockHeight)
		if err != nil {
			idx.Errorf("failed to call at %d height data, %s", prevBlockHeight, err)
			return lastIndexPointerEpoch, err
		}

		blockSummary := types.BlockSummary{BlockProposerAddress: preBlockProposerAddress, CosmosValidators: validators}

		// update validator address
		{
			isNewValidator := false
			newValidatorAddressMap := make(map[string]bool)
			_, exist := idx.Vim[blockSummary.BlockProposerAddress]
			if !exist {
				newValidatorAddressMap[blockSummary.BlockProposerAddress] = true
				isNewValidator = true
			}
			for _, validator := range blockSummary.CosmosValidators {
				_, exist := idx.Vim[validator.Address]
				if !exist {
					// idx.Debugf("the validator isn't in current validator info table: %s | %s", validator.Address, validator.Pubkey.Value)
					newValidatorAddressMap[validator.Address] = true
					isNewValidator = true
				}
			}

			// this logic will be progressed only when there are new tendermint validators in this block
			if isNewValidator {
				newValidatorInfoList, err := function.MakeValidatorInfoList(idx.CommonApp,
					idx.ChainID, idx.ChainInfoID,
					idx.ChainName, idx.IsConsumer,
					newValidatorAddressMap,
					prevBlockHeight)
				if err != nil {
					errors.Wrap(err, "failed to make validator info list")
				}

				// insert new validators' proposer address into the validator info table
				err = idx.repo.InsertValidatorInfoList(newValidatorInfoList)
				if err != nil {
					// NOTE: fetch again validator_info list, actually already inserted the list by other indexer service
					idx.FetchValidatorInfoList()
					return lastIndexPointerEpoch, errors.Wrap(err, "failed to insert new hex address list")
				}

				// get already saved tendermint validator list for mapping validators ids
				validatorInfoList, err := idx.repo.GetValidatorInfoListByChainInfoID(idx.ChainInfoID)
				if err != nil {
					return lastIndexPointerEpoch, errors.Wrap(err, "failed to get new validator info list after inserting new hex address list")

				}

				for _, validator := range validatorInfoList {
					idx.Vim[validator.HexAddress] = int64(validator.ID)
				}

				idx.Debugf("changed vim length: %d", len(idx.Vim))
			}

		}

		// get epoch data
		resp, err = idx.APIClient.Get(ctx, BlockTxsQueryPath(firstBlockHeightInEpoch))
		if err != nil {
			idx.Errorln(err)
			return lastIndexPointerEpoch, err
		}

		blockHeight, blockTimestamp, bexVotes, err := ExtractBabylonExtendVoteAndBlockInfo(resp)
		if err != nil {
			idx.Errorln(err)
			return lastIndexPointerEpoch, err
		}

		idx.Debugf("found %d babylon validators' BLS votes in the epoch+1 block", len(bexVotes))
		bveList, err := makeBabylonExtensionVote(
			// logger
			idx.Entry,
			// vms instance data
			idx.ChainInfoID,
			idx.Vim,
			// current block data
			blockHeight,
			blockTimestamp,
			epoch,
			bexVotes,
		)
		if err != nil {
			return lastIndexPointerEpoch, errors.Wrapf(err, "failed to make babylon vote extension list at %d epoch", epoch)
		}
		idx.Debugf("made %d bveList in the %d epoch", len(bveList), epoch)
		if len(bveList) > 0 {
			idx.Infof("found %d validators BLS signings in the %d epoch", len(bveList), epoch)
		}

		// NOTE: if solo validator mode, we don't need to insert all validotors vote status.
		// so, filter statues by moniker
		if len(idx.Monikers) > 0 {
			// if not init monikerIDMap
			if len(idx.MonikerIDMap) != len(idx.Monikers) {
				// init monikerIDMap
				validatorInfoList, err := idx.repo.GetValidatorInfoListByMonikers(idx.ChainInfoID, idx.Monikers)
				if err != nil {
					return lastIndexPointerEpoch, errors.Wrap(err, "failed to get validator_info list by monikers")
				}
				monikerIDMap := make(indexertypes.MonikerIDMap)
				for _, vi := range validatorInfoList {
					monikerIDMap[vi.ID] = true
				}
				// restore monikerIDMap in voteindexer struct, for reusing
				idx.MonikerIDMap = monikerIDMap
			}
			// override for solo validator
			bveList = filterValidatorVoteListByMonikers(idx.MonikerIDMap, bveList)
		}

		// need to save list and new pointer
		err = idx.repo.InsertBabylonVoteExtensionList(idx.ChainInfoID, firstBlockHeightInEpoch, bveList)
		if err != nil {
			return lastIndexPointerEpoch, err
		}

		// update epoch & metrics
		idx.updatePrometheusMetrics(epoch, blockSummary.BlockTimeStamp)
		idx.Debugf("updated babylon checkpoint BLS singings in %d epoch ", epoch)
		return epoch, nil
	}

	// NOTE: sync will be working by each epoch. so, the normal situation it's not gonna run in the code.
	return 0, errors.New("unexpected errors")
}

func makeBabylonExtensionVote(
	// vmc instance data
	vml *logrus.Entry,
	chainInfoID int64,
	validatorIDMap indexertypes.ValidatorIDMap,
	blockHeight int64,
	blockTimestamp time.Time,
	epoch int64,
	votes []BabylonExtendVote,
) ([]model.BabylonVoteExtension, error) {
	bevList := make([]model.BabylonVoteExtension, 0)
	for _, vote := range votes {
		status := int64(tmtypes.BlockIDFlag_value[vote.BlockIDFlag])
		hexAddress, err := base64.StdEncoding.DecodeString(vote.Validator.Address)
		if err != nil {
			return nil, errors.New("failed to make babylon extension vote")
		}

		validatorAddress := fmt.Sprintf("%X", hexAddress)
		if vote.ExtensionSignature == "" {
			vml.Debugf(`found miss validator idx: %d, address: %s in this %d height`, validatorIDMap[validatorAddress], validatorAddress, blockHeight)
		}

		validatorHexAddressID, exist := validatorIDMap[validatorAddress]
		if !exist {
			vml.Debugf("debug: %v", vote)
			return nil, errors.New("failed to find missed validators hex address id in validator id maps")
		}

		// for committed voters
		bevList = append(bevList, model.BabylonVoteExtension{
			ChainInfoID:           chainInfoID,
			Epoch:                 epoch,
			Height:                blockHeight,
			Timestamp:             blockTimestamp,
			ValidatorHexAddressID: validatorHexAddressID,
			Status:                status,
		})
	}

	return bevList, nil
}

func filterValidatorVoteListByMonikers(monikerIDMap indexertypes.MonikerIDMap, bevList []model.BabylonVoteExtension) []model.BabylonVoteExtension {
	// already inited monikerIDMap just filter validator vote by moniker id maps
	newValidatorVoteList := make([]model.BabylonVoteExtension, 0)
	for _, vv := range bevList {
		// // only append validaor vote in package monikers
		_, exist := monikerIDMap[vv.ValidatorHexAddressID]
		if exist {
			newValidatorVoteList = append(newValidatorVoteList, vv)
		}
	}
	return newValidatorVoteList
}
