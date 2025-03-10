package btcdelegation

import (
	"os"
	"testing"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/testutil"

	"github.com/stretchr/testify/assert"
)

func TestGetStatus(t *testing.T) {
	_ = testutil.SetupForTest()
	testMoniker := os.Getenv("TEST_MONIKER")
	testCases := []struct {
		testingName      string
		chainName        string
		protocolType     string
		chainID          string
		isConsumer       bool
		endpoint         common.Endpoints
		providerEndpoint common.Endpoints
		expectResult     float64
	}{
		{
			testingName:  "Babylon Chain",
			chainName:    "babylon",
			protocolType: "cosmos",
			isConsumer:   false,
			endpoint: common.Endpoints{
				RPCs: []string{os.Getenv("TEST_BABYLON_RPC_ENDPOINT")},
				APIs: []string{os.Getenv("TEST_BABYLON_API_ENDPOINT")},
			},
		},
	}

	for _, tc := range testCases {
		exporter := testutil.GetTestExporter()
		t.Run(tc.testingName, func(t *testing.T) {
			if !assert.NotEqualValues(t, tc.endpoint, "") {
				// endpoint is empty
				t.FailNow()
			}

			// setup
			exporter.SetRPCEndPoint(tc.endpoint.RPCs[0])
			exporter.SetAPIEndPoint(tc.endpoint.APIs[0])
			exporter.ChainName = tc.chainName

			_ = testMoniker
			// // start test
			// status, err := GetFinalityProviderUptime(exporter)
			// if err != nil {
			// 	t.Fatalf("%s: %s", tc.testingName, err.Error())
			// }

			// for _, fp := range status.FinalityProvidersStatus {
			// 	if fp.Moniker == testMoniker {
			// 		t.Log("fp moniker          :", fp.Moniker)
			// 		t.Log("fp operator address :", fp.Address)
			// 		t.Log("fp btc pk           :", fp.BTCPK)
			// 		t.Log("fp miss count       :", fp.MissedBlockCounter)
			// 		t.Log("fp miss count       :", fp.MissedBlockCounter)
			// 	}
			// }
		})
	}
}
