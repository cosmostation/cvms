package collector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/helper/config"
	tests "github.com/cosmostation/cvms/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func Test_UpgradeCollector(t *testing.T) {
	// setup
	_ = tests.SetupForTest()
	type testCase struct {
		testingName  string
		chainName    string
		protocolType string
		endpoints    common.Endpoints
	}

	testCases := []testCase{
		{
			testingName:  "Cosmos Upgrade Collector Test",
			chainName:    "cosmos",
			protocolType: "cosmos",
			endpoints: common.Endpoints{
				RPCs: []string{os.Getenv("TEST_UPGRADE_RPC_ENDPOINT")},
				APIs: []string{os.Getenv("TEST_UPGRADE_API_ENDPOINT")},
			},
		},
	}

	for _, tc := range testCases {
		cc := config.ChainConfig{
			DisplayName: tc.chainName,
		}
		// build packager
		packager, err := common.NewPackager(common.NETWORK, tests.TestFactory, tests.TestLogger, true, "chainid", tc.chainName, Subsystem, tc.protocolType, cc, tc.endpoints)
		if err != nil {
			t.Fatal(err)
		}
		err = Start(*packager)
		assert.NoErrorf(t, err, "error message %s", "formatted")
	}

	// sleep 3sec for waiting collecting data from nodes
	time.Sleep(10 * time.Second)

	// Create a new HTTP test server
	server := httptest.NewServer(tests.TestHandler)
	defer server.Close()

	// Make a request to the test server
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	// Check the HTTP response status code
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}

	// Check the response body for the expected metric
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	t.Logf("recevied:\n%s", body)

	// NOTE: only when there is onchain upgrade, you can check this logic
	// for _, tc := range testCases {
	// 	checkMetric := testMetricName(common.Namespace, Subsystem, RemainingTimeMetricName, tc.chainName, packageName)
	// 	if !strings.Contains(string(body), checkMetric) {
	// 		t.Fatalf("Expected metric '%s' not found in response body: %s", checkMetric, string(body))
	// 	} else {
	// 		t.Logf("Found expected metric: %s", checkMetric)
	// 	}
	// }
}
