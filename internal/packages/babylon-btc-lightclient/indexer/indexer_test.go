package indexer

import (
	"testing"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/api"

	"github.com/cosmostation/cvms/internal/helper/logger"
	"github.com/stretchr/testify/assert"
)

var (
	p = common.Packager{
		ChainName:    "babylon",
		ChainID:      "bbn-test-5",
		ProtocolType: "cosmos",
		Endpoints: common.Endpoints{
			// RPCs: []string{"https://rpc-office.cosmostation.io/babylon-testnet"},
			RPCs: []string{"https://babylon-testnet-rpc.polkachu.com"},
			APIs: []string{"https://lcd-office.cosmostation.io/babylon-testnet"},
		},
		Logger: logger.GetTestLogger(),
	}
)

func Test_1_GetBTCLightClientData(t *testing.T) {
	app := common.NewCommonApp(p)
	app.SetAPIEndPoint(p.Endpoints.APIs[0])
	app.SetRPCEndPoint(p.Endpoints.RPCs[0])

	_, err := api.GetBabylonBTCLightClientParams(app.CommonClient)
	assert.NoError(t, err)

	// 23921 : empty height
	// 23828 : btc forward event height
	// 23922 : btc roll back event height
	testHeights := []int64{23921, 23828, 23922}

	for _, h := range testHeights {
		txsEvents, _, err := api.GetBlockResults(app.CommonClient, h)
		assert.NoError(t, err)

		// NOTE: bieList means btc insert events list
		bieList, err := filterBTCLightClientEvents(txsEvents)
		assert.NoError(t, err)

		for _, bie := range bieList {
			t.Logf("len header: %d", len(bie.BTCHeaders))
			t.Logf("in %d height, got %s headers", h, bie.ToHeadersStringSlice())
		}
	}
}

func Test_2_BatchSync(t *testing.T) {
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	p.SetIndexerDB(indexerDB)
	p.IsConsumerChain = false
	assert.NoError(t, err)

	idx, err := NewBTCLightClientIndexer(p)
	assert.NoError(t, err)

	err = idx.InitChainInfoID()
	assert.NoError(t, err)

	err = idx.InitPartitionTablesByChainInfoID(idx.IndexName, idx.ChainID, 100)
	assert.NoError(t, err)

	err = idx.FetchValidatorInfoList()
	assert.NoError(t, err)

	newIndexPointer, err := idx.batchSync(23920)
	assert.NoError(t, err)
	t.Logf("new index point: %d", newIndexPointer)
}

func Test_3_Start(t *testing.T) {
	waitingDuration := 60 * time.Second
	// Step 1: Set up the database
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	assert.NoError(t, err)

	p.SetIndexerDB(indexerDB)
	p.SetRetentionTime("1h")
	p.IsConsumerChain = false

	// Step 2: Initialize the CheckpointIndexer
	idx, err := NewBTCLightClientIndexer(p)
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

func Test_4_ValidationLogic(t *testing.T) {
	app := common.NewCommonApp(p)
	app.SetAPIEndPoint(p.Endpoints.APIs[0])
	app.SetRPCEndPoint(p.Endpoints.RPCs[0])

	_, err := api.GetBabylonBTCLightClientParams(app.CommonClient)
	assert.NoError(t, err)

	testHeight := int64(261439)
	expectedReporter := "bbn1mzghl5csl75wz86e70j6ggdll4huazgfmeucyx"
	txsEvents, _, err := api.GetBlockResults(app.CommonClient, testHeight)
	assert.NoError(t, err)

	// NOTE: bieList means btc insert events list
	bieList, err := filterBTCLightClientEvents(txsEvents)
	assert.NoError(t, err)

	for _, bie := range bieList {
		t.Logf("len header: %d", len(bie.BTCHeaders))
		t.Logf("in %d height, got %s headers", testHeight, bie.ToHeadersStringSlice())
		assert.Equal(t, expectedReporter, bie.ReporterAddress)
	}
}
