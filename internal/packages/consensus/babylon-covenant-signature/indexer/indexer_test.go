package indexer

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/helper/logger"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-covenant-signature/repository"

	"github.com/stretchr/testify/assert"
)

var (
	syncStartHeight int64 = 278639

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

func TestEmptyTxsBlock(t *testing.T) {
	app := common.NewCommonApp(p)
	app.SetAPIEndPoint(p.Endpoints.APIs[0])
	app.SetRPCEndPoint(p.Endpoints.RPCs[0])

	testHeight := int64(17293874)
	height, timestamp, txs, err := api.GetBlockAndTxs(app.CommonClient, testHeight)
	assert.NoError(t, err)

	fmt.Println(height, timestamp, txs)
}

func TestLogic(t *testing.T) {
	app := common.NewCommonApp(p)
	app.SetAPIEndPoint(p.Endpoints.APIs[0])
	app.SetRPCEndPoint(p.Endpoints.RPCs[0])

	testHeight := int64(278894)
	_, blockTimestamp, _, _, _, _, err := api.GetBlock(app.CommonClient, testHeight)
	assert.NoError(t, err)

	txsEvents, _, err := api.GetBlockResults(app.CommonClient, testHeight)
	assert.NoError(t, err)

	covenantSigEvents := make([]EventCovenantSignature, 0)
	btcDelegationEvents := make([]EventBtcDelegationCreated, 0)

	for _, event := range txsEvents {
		supportEvent, err := ParseDynamicEvent(event)
		if errors.Is(err, common.ErrUnSupportedEventType) {
			continue
		} else {
			assert.NoError(t, err)
		}

		if e, ok := supportEvent.(EventCovenantSignature); ok {
			covenantSigEvents = append(covenantSigEvents, e)
		} else if e, ok := supportEvent.(EventBtcDelegationCreated); ok {
			btcDelegationEvents = append(btcDelegationEvents, e)
		}
	}

	fmt.Println("Block Height: ", testHeight)
	fmt.Println("Block Timestamp: ", blockTimestamp.Unix())
	fmt.Println("========= Covenant Signature Events ========== Total:", len(covenantSigEvents))
	for _, e := range covenantSigEvents {
		fmt.Println("Covenant Btc Pk: ", e.CovenantBtcPkHex)
		fmt.Println("Covenant Unbonding Sig: ", e.CovenantUnbondingSignature)
		fmt.Println("Staing Tx Hash: ", e.StakingTxHash)
		fmt.Println("-")
	}
	fmt.Println("========= BTC Delegation Events ========== Total:", len(btcDelegationEvents))
	for _, e := range btcDelegationEvents {
		escape, err := DecodeEscapedJSONString(e.StakingTxHash)
		assert.NoError(t, err)

		hash, err := DecodeBTCStakingTxByHexStr(escape)
		assert.NoError(t, err)
		fmt.Println("Staing Tx Hash: ", hash)
		fmt.Println("-")
	}

	assert.Equal(t, blockTimestamp.Unix(), int64(1739364888))

	// Success: 6, failed: 2(out of gas...)
	assert.Equal(t, len(covenantSigEvents), 6)
	assert.Equal(t, len(btcDelegationEvents), 1)
}

func TestStart(t *testing.T) {
	waitingDuration := 60 * time.Second
	// Step 1: Set up the database
	tempDBName := "cvms"
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

	err = idx.csRepo.InitPartitionTablesByChainInfoID(repository.IndexName, idx.ChainID, 100)
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
