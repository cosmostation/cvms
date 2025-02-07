package indexer

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper/logger"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-checkpoint/repository"

	"github.com/stretchr/testify/assert"
)

var (
	p = common.Packager{
		ChainName:    "babylon",
		ChainID:      "bbn-test-5",
		ProtocolType: "cosmos",
		Endpoints: common.Endpoints{
			RPCs: []string{"https://rpc-office.cosmostation.io/babylon-testnet"},
			APIs: []string{"https://lcd-office.cosmostation.io/babylon-testnet"},
		},
		Logger: logger.GetTestLogger(),
	}
)

func TestStart(t *testing.T) {
	waitingDuration := 60 * time.Second
	// Step 1: Set up the database
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	assert.NoError(t, err)

	p.SetIndexerDB(indexerDB)
	p.SetRetentionTime("1h")
	p.IsConsumerChain = false

	// Step 2: Initialize the CheckpointIndexer
	idx, err := NewCheckpointIndexer(p)
	assert.NoError(t, err)

	// Modify Start() to accept a callback for testing purposes
	err = idx.Start()
	assert.NoError(t, err)

	// Step 4: Wait for the goroutine to finish
	done := make(chan bool)

	go func() {
		// Simulate some work
		time.Sleep(waitingDuration)
		done <- true
	}()

	<-done
	t.Log("Goroutine finished work")
}
func TestBatchSync(t *testing.T) {
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	p.SetIndexerDB(indexerDB)
	p.IsConsumerChain = false
	assert.NoError(t, err)

	idx, err := NewCheckpointIndexer(p)
	assert.NoError(t, err)

	err = idx.InitChainInfoID()
	assert.NoError(t, err)

	err = idx.repo.InitPartitionTablesByChainInfoID(repository.IndexName, idx.ChainID, 100)
	assert.NoError(t, err)

	err = idx.FetchValidatorInfoList()
	assert.NoError(t, err)

	newIndexPointer, err := idx.batchSync(1)
	assert.NoError(t, err)
	t.Logf("new index point: %d", newIndexPointer)
}

func TestSyncEpoch(t *testing.T) {
	p.SetIndexerDB(&common.IndexerDB{})
	idx, err := NewCheckpointIndexer(p)
	assert.NoError(t, err)

	ctx := context.Background()
	uri := "/babylon/epoching/v1/current_epoch"
	resp, err := idx.APIClient.Get(ctx, uri)
	assert.NoErrorf(t, err, "error from : %s", uri)

	// pasrse
	var currentEpochResponse CurrentEpochResponse
	err = json.Unmarshal(resp, &currentEpochResponse)
	assert.NoError(t, err)

	currentEpoch, err := strconv.Atoi(currentEpochResponse.CurrentEpoch)
	assert.NoError(t, err)

	// for loop until current epoch
	for epoch := int64(0); epoch <= int64(currentEpoch); epoch++ {
		if epoch == 0 {
			t.Log("skip the zero epoch")
			continue
		}
		path := fmt.Sprintf("/babylon/epoching/v1/epochs/%d", epoch)
		resp, err := idx.APIClient.Get(ctx, path)
		assert.NoError(t, err)

		// pasre type
		EpochsResponse := EpochResponse{}
		err = json.Unmarshal(resp, &EpochsResponse)
		assert.NoError(t, err)

		// parse
		t.Logf("epoch number: %s | first block height: %s ", EpochsResponse.Epoch.EpochNumber, EpochsResponse.Epoch.FirstBlockHeight)

		firstBlockOfEpoch, err := strconv.Atoi(EpochsResponse.Epoch.FirstBlockHeight)
		assert.NoError(t, err)

		path = fmt.Sprintf("/cosmos/tx/v1beta1/txs/block/%d?pagination.limit=1", firstBlockOfEpoch)
		// get first tx in the first block in the each epoch
		resp, err = idx.APIClient.Get(ctx, path)
		assert.NoError(t, err)

		// parse
		// t.Logf("first tx in the block: %s", resp.Body())

		// pasre type
		txsReponse := BlockTxsResponse{}
		err = json.Unmarshal(resp, &txsReponse)
		assert.NoError(t, err)

		for _, tx := range txsReponse.Txs {
			for _, message := range tx.Body.Messages {
				var preResult map[string]json.RawMessage
				if err := json.Unmarshal(message, &preResult); err != nil {
					t.Errorf("unexpected error: %s", err)
				}
				if rawType, ok := preResult["@type"]; ok {
					var typeValue string
					if err := json.Unmarshal(rawType, &typeValue); err != nil {
						t.Errorf("unexpected error parsing @type: %s", err)
					}
					votes, err := ParseDynamicMessage(message, typeValue)
					if err != nil {
						t.Errorf("unexpected error parsing message: %s", err)
					}

					t.Logf("votes: %v", votes)
				} else {
					panic("@type key is missing")
				}
			}
		}

		if epoch == 2 {
			break
		}
	}
}

// check the decoded address is same on cometbft rpc validator address
func TestSyncMachnismForBLSsigning(t *testing.T) {
	address := "YgV0dg2QBEHvkQlvhIV3Wz3RwKY="
	hexAddress, _ := base64.StdEncoding.DecodeString(address)
	validatorAddress := "620574760D900441EF91096F8485775B3DD1C0A6"
	if !(validatorAddress == fmt.Sprintf("%X", hexAddress)) {
		t.Errorf("validator address is not correct")
	} else {
		t.Logf("validator address is correct: %X", hexAddress)
	}
}
func TestMakeBLSAddressFromCometBFTAddress(t *testing.T) {
	hexAddress := "96FD92D649AE03B2F5B83098D3E47669F99200B5"

	// Decode the hex string to bytes
	bytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		t.Errorf("failed to decode hex string: %v", err)
	}

	// Encode the bytes to a base64 string
	base64Address := base64.StdEncoding.EncodeToString(bytes)

	// Expected base64-encoded string
	expectedAddress := "lv2S1kmuA7L1uDCY0+R2afmSALU="

	// Validate the result
	if base64Address != expectedAddress {
		t.Errorf("base64 address does not match: got %s, expected %s", base64Address, expectedAddress)
	} else {
		t.Logf("base64 address is correct: %s", base64Address)
	}

}
