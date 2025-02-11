package indexer

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/helper/logger"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-covenant-signature/model"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-covenant-signature/repository"

	"github.com/stretchr/testify/assert"
)

var (
	syncStartHeight int64 = 245219

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

func TestLogic2(t *testing.T) {
	str := "AgAAAAEV+HqXcrYPtvInS3ygpwH2bK/I4JhHdNMBbmh6ht3sTgAAAAAA/////wJQwwAAAAAAACJRIHitg+6DT820A8/DE8LWeEN6BFZiPpMkj4URXmap+uAOfXYOAAAAAAAiUSAtyerAPfEAfA5g4Vn/341CdaYRMelp7cwDvkKfNnMFDgAAAAA="
	t.Logf("raw base64: %s", str)

	hexBz, err := base64.StdEncoding.DecodeString(str)
	assert.NoError(t, err)
	t.Logf("raw hex: %X", hexBz)

	// Deserialize transaction
	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(hexBz))
	if err != nil {
		log.Fatal(err)
	}

	// Print decoded transaction
	fmt.Printf("txId: %s\n", tx.TxHash().String())

}

func TestLogic(t *testing.T) {
	app := common.NewCommonApp(p)
	app.SetAPIEndPoint(p.Endpoints.APIs[0])
	app.SetRPCEndPoint(p.Endpoints.RPCs[0])

	testHeight := int64(245259)
	height, timestamp, txs, err := api.GetBlockAndTxs(app.CommonClient, testHeight)
	assert.NoError(t, err)

	newMsgCovenantSigs := make([]MsgCovenantSignature, 0)
	for _, tx := range txs {
		for _, message := range tx.Body.Messages {
			var preResult map[string]json.RawMessage
			err := json.Unmarshal(message, &preResult)
			assert.NoError(t, err)

			if rawType, ok := preResult["@type"]; ok {
				var typeValue string
				err := json.Unmarshal(rawType, &typeValue)
				assert.NoError(t, err)

				covenantSigs, err := ParseDynamicMessage(message, typeValue)
				if errors.Is(err, common.ErrUnSupportedMessageType) {
					continue
				} else {
					assert.NoError(t, err)
				}
				newMsgCovenantSigs = append(newMsgCovenantSigs, covenantSigs)
			}
		}
	}

	t.Logf("got %d messages", len(newMsgCovenantSigs))

	var modelList = make([]model.BabylonCovenantSignature, 0)
	for idx, sig := range newMsgCovenantSigs {

		// It's not yet clear if Committee members can change dynamically, we've added some temporary code to prevent panic
		// pkID, exists := idx.covenantCommitteeMap[sig.Pk]
		// if !exists {
		// 	idx.Errorf("Missing covenant committee entry for PK: %s", sig.Pk)
		// 	continue
		// }

		newCovenantSignature := model.BabylonCovenantSignature{
			ChainInfoID:      1,
			Height:           height,
			CovenantBtcPkID:  int64(idx),
			BTCStakingTxHash: sig.StakingTxHash,
			Timestamp:        timestamp,
		}

		modelList = append(modelList, newCovenantSignature)
	}

	t.Logf("got %d model list", len(modelList))
	for _, model := range modelList {
		t.Logf("%s", model)
	}
}

func TestStart(t *testing.T) {
	waitingDuration := 60 * time.Second
	// Step 1: Set up the database
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	assert.NoError(t, err)

	p.SetIndexerDB(indexerDB)
	p.SetRetentionTime("1h")
	p.IsConsumerChain = false

	// Step 2: Initialize the CovenantSignatureIndexer
	idx, err := NewCovenantSignatureIndexer(p)
	assert.NoError(t, err)

	//test target height
	idx.earliestBlockHeight = syncStartHeight

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
	waitingDuration := 60 * time.Second
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	p.SetIndexerDB(indexerDB)
	p.IsConsumerChain = false
	assert.NoError(t, err)

	idx, err := NewCovenantSignatureIndexer(p)
	assert.NoError(t, err)

	err = idx.InitChainInfoID()
	assert.NoError(t, err)

	err = idx.repo.InitPartitionTablesByChainInfoID(repository.IndexName, idx.ChainID, 100)
	assert.NoError(t, err)

	// Step 4: Wait for the goroutine to finish
	done := make(chan bool)

	newIndexPointer, err := idx.batchSync(idx.earliestBlockHeight, idx.earliestBlockHeight+1)

	go func() {
		// Simulate some work
		time.Sleep(waitingDuration)
		done <- true
	}()

	assert.NoError(t, err)
	t.Logf("new index point: %d", newIndexPointer)
}
