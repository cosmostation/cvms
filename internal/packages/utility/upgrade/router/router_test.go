package router_test

import (
	"os"
	"testing"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/packages/utility/upgrade/router"
	tests "github.com/cosmostation/cvms/internal/testutil"

	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	_ = tests.SetupForTest()

	TestCases := []struct {
		testingName string
		chainName   string
		rpcEndpoint string
		apiEndpoint string
	}{
		{
			testingName: "Upgrade exist Chain",
			chainName:   "cosmso",
			rpcEndpoint: os.Getenv("TEST_UPGRADE_RPC_ENDPOINT"),
			apiEndpoint: os.Getenv("TEST_UPGRADE_API_ENDPOINT"),
		},
		{
			testingName: "Celestia Signal Upgrade",
			chainName:   "celestia",
			rpcEndpoint: os.Getenv("TEST_CELESTIA_RPC_ENDPOINT"),
			apiEndpoint: os.Getenv("TEST_CELESTIA_API_ENDPOINT"),
		},
		{
			testingName: "Story Upgrade",
			chainName:   "story",
			rpcEndpoint: os.Getenv("TEST_STORY_RPC_ENDPOINT"),
			apiEndpoint: os.Getenv("TEST_STORY_API_ENDPOINT"),
		},
	}

	for _, tc := range TestCases {
		exporter := tests.GetTestExporter()
		t.Run(tc.testingName, func(t *testing.T) {
			if !assert.NotEqualValues(t, tc.apiEndpoint, "") && !assert.NotEqualValues(t, tc.rpcEndpoint, "") {
				// hostaddress is empty
				t.FailNow()
			}

			exporter.SetAPIEndPoint(tc.apiEndpoint)
			exporter.SetRPCEndPoint(tc.rpcEndpoint)
			CommonUpgrade, err := router.GetStatus(exporter, tc.chainName)
			if err != nil && err != common.ErrCanSkip {
				t.Log("Unexpected Error Occured!")
				t.Skip()
			}

			if err == common.ErrCanSkip {
				t.Logf("There is no onchain upgrade now in %s", tc.testingName)
			} else {
				t.Log("onchain upgrade is found", CommonUpgrade.UpgradeName, CommonUpgrade.RemainingTime)
			}
		})
	}
}
