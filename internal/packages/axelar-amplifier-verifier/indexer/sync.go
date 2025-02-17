package indexer

import (
	"sync"
	"time"

	"github.com/cosmostation/cvms/internal/common/api"
	indexermodel "github.com/cosmostation/cvms/internal/common/indexer/model"
	indexertypes "github.com/cosmostation/cvms/internal/common/indexer/types"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/axelar-amplifier-verifier/model"
	"github.com/pkg/errors"
)

const blockExpiry = 10

func (idx *AxelarAmplifierVerifierIndexer) batchSync(lastIndexPoint int64) (
	/* new index pointer */ int64,
	/* error */ error,
) {
	// set starntHeight and endHeight for batch sync
	// NOTE: end height will use latest height - 10 for block expiry height
	startHeight := (lastIndexPoint + 1)
	endHeight := (idx.Lh.LatestHeight - blockExpiry)
	if startHeight > endHeight {
		idx.Infof("no need to sync from %d height to %d height, so it'll skip the logic", startHeight, endHeight)
		return lastIndexPoint, nil
	}

	// set limit at end-height in this batch sync logic
	if (endHeight - startHeight) > indexertypes.BatchSyncLimit {
		endHeight = startHeight + indexertypes.BatchSyncLimit
		idx.Infof("by batch sync limit, end height will change to %d", endHeight)
	}

	// get contract info
	chainNameMap, err := GetVerifierContractAddressMap(idx.CommonClient, false)
	if err != nil {
		return lastIndexPoint, errors.Wrap(err, "failed get verifier register contract address")
	}

	// init channel and waitgroup for go-routine
	ch := make(chan helper.Result)
	wg := sync.WaitGroup{}
	summary := make(map[int64]PollDataSummary)

	// start to call block results
	for h := startHeight; h <= endHeight; h++ {
		wg.Add(1)
		height := h

		go func(ch chan helper.Result) {
			defer helper.HandleOutOfNilResponse(idx.Entry)
			defer wg.Done()

			txsEvents, _, err := api.GetBlockResults(idx.CommonClient, height)
			if err != nil {
				idx.Errorf("failed to call at %d height block results, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			if len(txsEvents) == 0 {
				polls := AmplifierPollStartFillter(txsEvents)
				ch <- helper.Result{
					Item: PollDataSummary{
						height:    height,
						polls:     polls,
						pollVotes: nil,
					},
					Success: true,
				}
				return
			}

			_, _, txs, err := api.GetBlockAndTxs(idx.CommonClient, height)
			if err != nil {
				idx.Errorf("failed to call at %d height block and txs, %s", height, err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			pollVotes, err := ExtractPoll(txs)
			if err != nil {
				idx.Errorln(err)
				ch <- helper.Result{Item: nil, Success: false}
				return
			}

			// idx.Debugf("got poll votes: %d in %d", len(pollVotes), height)

			polls := AmplifierPollStartFillter(txsEvents)
			ch <- helper.Result{
				Item: PollDataSummary{
					height:    height,
					polls:     polls,
					pollVotes: pollVotes,
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
			item := r.Item.(PollDataSummary)
			summary[item.height] = item
			continue
		}
		errorCount++
	}

	// check error count
	if errorCount > 0 {
		return lastIndexPoint, errors.Errorf("failed to collect batch poll data, total errors: %d", errorCount)
	}

	// first add new verifiers
	isNewVerifier := false
	newVerifierInfo := make([]indexermodel.VerifierInfo, 0)
	for h := startHeight; h <= endHeight; h++ {
		for _, poll := range summary[h].polls {
			for _, verifier := range poll.Participants {
				_, exist := idx.Vim[verifier]
				if !exist {
					idx.Debugf("the %s isn't in current verifier info table, the address will be added into the meta table", verifier)
					isNewVerifier = true
					newVerifierInfo = append(newVerifierInfo, indexermodel.VerifierInfo{
						ChainInfoID:     idx.ChainInfoID,
						VerifierAddress: verifier,
						Moniker:         verifier,
					})
				}
			}
		}
	}

	if isNewVerifier {
		idx.Debugf("insert new amplifier verifiers: %d", len(newVerifierInfo))
		err := idx.InsertVerifierInfoList(newVerifierInfo)
		if err != nil {
			// NOTE: fetch again validator_info list, actually already inserted the list by other indexer service
			idx.FetchValidatorInfoList()
			return lastIndexPoint, errors.Wrap(err, "failed to insert new reporter list")
		}

		verifierInfoList, err := idx.GetVerifierInfoListByChainInfoID(idx.ChainInfoID)
		if err != nil {
			return lastIndexPoint, errors.Wrap(err, "failed to get new reporter info list after inserting new hex address list")

		}

		for _, v := range verifierInfoList {
			idx.Vim[v.VerifierAddress] = int64(v.ID)
			idx.VAM[v.ID] = v.VerifierAddress
		}

		idx.Debugf("changed vim length: %d and VAM: %d", len(idx.Vim), len(idx.VAM))
	}

	// first key: contract address
	// second key: verifier address
	pollMap := make(PollMap)
	for h := startHeight; h <= endHeight; h++ {
		idx.Debugf("there are %d polls in %d height", len(summary[h].polls), h)
		for _, poll := range summary[h].polls {
			// ✅ Ensure pollMap[contractAddress] exists before assigning a value
			chainAndPollID := ConcatChainAndPollID(poll.SourceChain, poll.PollID)
			if _, exists := pollMap[chainAndPollID]; !exists {
				pollMap[chainAndPollID] = make(map[string]model.AxelarAmplifierVerifierVote)
			}

			idx.Debugf("%s poll start with %d verifiers", chainAndPollID, len(poll.Participants))
			for _, verifier := range poll.Participants {
				initVote := model.AxelarAmplifierVerifierVote{
					// ID: Autoincrement
					ChainInfoID:     idx.ChainInfoID,
					CreatedAt:       time.Now(),
					ChainAndPollID:  chainAndPollID,
					PollStartHeight: h,
					PollVoteHeight:  0,
					VerifierID:      idx.Vim[verifier],
					Status:          model.PollStart,
				}
				pollMap[chainAndPollID][verifier] = initVote
				idx.Debugf("[%s] was inited for %s", chainAndPollID, verifier)
			}
		}

		cnt := 0
		for _, pv := range summary[h].pollVotes {
		RetryAfterInitPoll:
			contractInfo, exist := chainNameMap[pv.ContractAddress]
			if !exist {
				return lastIndexPoint, errors.Wrap(err, "unexpected poll voted was occured")
			}

			key := ConcatChainAndPollID(contractInfo.ChainName, pv.PollID)
			// Ensure the outer map exists
			if _, exists := pollMap[key]; !exists {
				idx.Infof("Poll key %s not found, try to get init poll votes", key)
				initVoteMap := idx.MustInitPoll(key, pv.ContractAddress, pv.PollID, int64(contractInfo.BlockExpiry))
				idx.Infof("got %d init votes", len(initVoteMap))
				pollMap[key] = initVoteMap
				goto RetryAfterInitPoll
			}

			// Ensure the inner map contains the verifier
			vote, exists := pollMap[key][pv.VerifierAddress]
			if !exists {
				idx.Errorf("Verifier %s not found under poll %s, skipping update.", pv.VerifierAddress, key)
				continue
			}

			// Update the status
			vote.UpdateVote(pv.StatusStr, h)

			// Store the updated item back in the map
			pollMap[key][pv.VerifierAddress] = vote
			cnt++
		}

		idx.Debugf("there are %d poll votes in %d height", cnt, h)
	}

	pollVoteList := ConvertPollMapToList(pollMap)
	idx.Infof("got total %d poll vote list from %d to %d", len(pollVoteList), startHeight, endHeight)
	err = idx.InsertValidatorExtensionVoteList(idx.ChainInfoID, endHeight, pollVoteList)
	if err != nil {
		return lastIndexPoint, errors.Wrap(err, "faeild to insert models")
	}

	idx.updatePrometheusMetrics(endHeight, pollMap)
	return endHeight, nil
}

type PollDataSummary struct {
	height    int64
	polls     []Poll
	pollVotes []PollVote
}

func (idx *AxelarAmplifierVerifierIndexer) MustInitPoll(chainAndPollID, contractAddress, pollID string, blockExpiry int64) map[string]model.AxelarAmplifierVerifierVote {
	expireHeight, participants, err := GetPollState(idx.CommonClient, contractAddress, pollID)
	if err != nil {
		return nil
	}

	isNewVerifier := false
	newVerifierInfo := make([]indexermodel.VerifierInfo, 0)
	for _, verifier := range participants {
		_, exist := idx.Vim[verifier]
		if !exist {
			idx.Debugf("the %s isn't in current verifier info table, the address will be added into the meta table", verifier)
			isNewVerifier = true
			newVerifierInfo = append(newVerifierInfo, indexermodel.VerifierInfo{
				ChainInfoID:     idx.ChainInfoID,
				VerifierAddress: verifier,
				Moniker:         verifier,
			})
		}
	}

	if isNewVerifier {
		idx.Debugf("insert new amplifier verifiers: %d", len(newVerifierInfo))
		err := idx.InsertVerifierInfoList(newVerifierInfo)
		if err != nil {
			return nil
		}

		err = idx.FetchValidatorInfoList()
		if err != nil {
			return nil
		}

		verifierInfoList, err := idx.GetVerifierInfoListByChainInfoID(idx.ChainInfoID)
		if err != nil {
			return nil
		}

		for _, v := range verifierInfoList {
			idx.Vim[v.VerifierAddress] = int64(v.ID)
			idx.VAM[v.ID] = v.VerifierAddress
		}
	}

	initVoteMap := make(map[string]model.AxelarAmplifierVerifierVote, len(participants))
	pollStartHeight := expireHeight - blockExpiry
	for _, verifier := range participants {
		initVote := model.AxelarAmplifierVerifierVote{
			// ID: Autoincrement
			ChainInfoID:     idx.ChainInfoID,
			CreatedAt:       time.Now(),
			ChainAndPollID:  chainAndPollID,
			PollStartHeight: pollStartHeight,
			PollVoteHeight:  0,
			VerifierID:      idx.Vim[verifier],
			Status:          model.PollStart,
		}
		initVoteMap[verifier] = initVote
	}

	return initVoteMap
}
