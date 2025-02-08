package indexer

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

func ParseCurrentEpoch(resp []byte) (int64, int64, error) {
	var currentEpochResponse CurrentEpochResponse
	err := json.Unmarshal(resp, &currentEpochResponse)
	if err != nil {
		return 0, 0, err
	}

	currentEpoch, err := strconv.ParseInt(currentEpochResponse.CurrentEpoch, 10, 64)
	if err != nil {
		return 0, 0, err

	}
	epochBoundaryHeight, err := strconv.ParseInt(currentEpochResponse.EpochBoundary, 10, 64)
	if err != nil {
		return 0, 0, err

	}

	return currentEpoch, epochBoundaryHeight, nil
}

// /* first block height by epoch */ int64,
// /* current epoch interval */ int64,
// /* unexpected error */ error,
func ParseEpoch(resp []byte) (
	/* first block height by epoch */ int64,
	/* current epoch interval */ int64,
	/* unexpected error */ error,
) {
	var result EpochResponse
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return 0, 0, err
	}

	firstBlockHeight, err := strconv.ParseInt(result.Epoch.FirstBlockHeight, 10, 64)
	if err != nil {
		return 0, 0, err

	}
	currentEpochInterval, err := strconv.ParseInt(result.Epoch.CurrentEpochInterval, 10, 64)
	if err != nil {
		return 0, 0, err

	}

	return firstBlockHeight, currentEpochInterval, nil
}

func ExtractBabylonExtendVoteAndBlockInfo(resp []byte) (
	/* block height */ int64,
	/* block timestamp */ time.Time,
	[]BabylonExtendVote,
	error,
) {
	result := BlockTxsResponse{}
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return 0, time.Time{}, nil, err
	}

	for _, tx := range result.Txs {
		for _, message := range tx.Body.Messages {
			var preResult map[string]json.RawMessage
			if err := json.Unmarshal(message, &preResult); err != nil {
				return 0, time.Time{}, nil, err
			}

			if rawType, ok := preResult["@type"]; ok {
				var typeValue string
				if err := json.Unmarshal(rawType, &typeValue); err != nil {
					return 0, time.Time{}, nil, err
				}

				votes, err := ParseDynamicMessage(message, typeValue)
				if err != nil {
					return 0, time.Time{}, nil, err
				}

				blockHeight, err := strconv.ParseInt(result.Block.Header.Height, 10, 64)
				if err != nil {
					return 0, time.Time{}, nil, err
				}

				return blockHeight, result.Block.Header.Time, votes, nil
			}
		}
	}

	return 0, time.Time{}, nil, errors.New("unexpected errors")
}

// ParseDynamicMessage dynamically parses the message based on its type.
func ParseDynamicMessage(message json.RawMessage, typeURL string) ([]BabylonExtendVote, error) {
	switch typeURL {
	case BabylonInjectedCheckpointMessageType:
		var msg MsgInjectedCheckpoint
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse MsgInjectedCheckpoint: %v", err)
			return nil, err
		}
		votes := append([]BabylonExtendVote{}, msg.ExtendedCommitInfo.Votes...)
		return votes, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", typeURL)
	}
}
