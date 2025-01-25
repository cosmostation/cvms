package helper_test

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/cosmostation/cvms/internal/helper"
	sdkhelper "github.com/cosmostation/cvms/internal/helper/sdk"
	tests "github.com/cosmostation/cvms/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	_ = tests.SetupForTest()
	os.Exit(m.Run())
}

// GetChainID Function Test
func Test_GetChainID(t *testing.T) {

	testCases := []struct {
		endpoints    []string
		protocolType string
	}{

		{
			[]string{
				os.Getenv("TEST_COSMOS_ENDPOINT_1"),
				os.Getenv("TEST_COSMOS_ENDPOINT_2"),
				os.Getenv("TEST_COSMOS_ENDPOINT_3"),
				os.Getenv("TEST_COSMOS_ENDPOINT_4"),
			},
			"cosmos",
		},

		// {
		// 	[]string{os.Getenv("TEST_ETHEREUM_HOST_ADDRESS")},
		// 	"cosmos",
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.protocolType, func(t *testing.T) {
			status := helper.GetOnChainStatus(tc.endpoints, tc.protocolType)
			assert.NotEqual(t, status.ChainID, "")
			assert.NotEqual(t, status.BlockHeight, 0)

			t.Logf("got onchain status: %v", status)
		})
	}

}

func Test_UnsetHttpURI(t *testing.T) {
	testCases := []struct {
		testURL             string `validate:"required,url"`
		expectedHostAddress string
		success             bool
	}{
		{
			testURL:             "http://sommelier.example.com:1337",
			expectedHostAddress: "sommelier.example.com:1337",
			success:             true,
		},
		{
			testURL:             "https://sommelier.example.com:1337",
			expectedHostAddress: "sommelier.example.com:1337",
			success:             true,
		},
		{
			testURL:             "grpc.example.com:1337",
			expectedHostAddress: "grpc.example.com:1337",
			success:             true,
		},
		{
			testURL:             "http:/sommelier.example.com:1337",
			expectedHostAddress: "sommelier.example.com:1337",
			success:             false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testURL, func(t *testing.T) {
			result, err := helper.UnsetHttpURI(tc.testURL)
			if tc.success && !assert.NoError(t, err) {
				assert.Equal(t, tc.expectedHostAddress, result)
				t.Log(err)
			}

			if !tc.success {
				// expected error
				assert.Error(t, err)
				t.SkipNow()
			}

			assert.NotEmpty(t, result)
			assert.Equal(t, tc.expectedHostAddress, result)
		})
	}

}

// NOTE: not yet validation by union team
func Test_KeyParsingForBn254(t *testing.T) {
	// {
	// 	"address": "02F071DC8D82DF56C497949356DF8BB13B595E3B",
	// 	"pub_key": {
	// 	"type": "tendermint/PubKeyBn254",
	// 	"value": "2BJTh+R6GW79g2QdJMIiSyC6+wjN/3/a+dbxiK1hpgY="
	// 	},
	// 	"voting_power": "1284624",
	// 	"proposer_priority": "-5125091"
	// 	},
	pubkey := "2BJTh+R6GW79g2QdJMIiSyC6+wjN/3/a+dbxiK1hpgY="
	decodedKey, _ := base64.StdEncoding.DecodeString(pubkey)
	addr, _ := sdkhelper.MakeProposerAddress(sdkhelper.Ed25519, decodedKey)
	t.Logf("parsed cons address: %s", addr)
}
func Test_KeyParsingForEd25519(t *testing.T) {
	pubkey := "e3BehnEIlGUAnJYn9V8gBXuMh4tXO8xxlxyXD1APGyk="
	decodedKey, _ := base64.StdEncoding.DecodeString(pubkey)
	addr, _ := sdkhelper.MakeProposerAddress(sdkhelper.Ed25519, decodedKey)
	t.Logf("parsed cons address: %s", addr)
}
func Test_KeyParsingForSecp256k1(t *testing.T) {
	// {
	// 	"address": "31A2EEB19577474467C3E21773D4B6D944A0042A",
	// 	"pub_key": {
	// 	"type": "tendermint/PubKeySecp256k1",
	// 	"value": "AmaxLjn8hMwPchbw9EFkYJmYaeXQmeRPIRbq/6fATLmO"
	// 	},
	// 	"voting_power": "3772204",
	// 	"proposer_priority": "-18745684"
	// 	},

	// "operator_address": "bandvaloper17d3uhcjlh4jyqcep82jfsrg8avsngklxxan5tg",
	// "consensus_pubkey": {
	// "@type": "/cosmos.crypto.secp256k1.PubKey",
	// "key": "AmaxLjn8hMwPchbw9EFkYJmYaeXQmeRPIRbq/6fATLmO"
	// },

	pubkey := "AmaxLjn8hMwPchbw9EFkYJmYaeXQmeRPIRbq/6fATLmO"
	decodedKey, _ := base64.StdEncoding.DecodeString(pubkey)

	t.Logf("hex %x", decodedKey)
	addr, _ := sdkhelper.MakeProposerAddress(sdkhelper.Secp256k1, decodedKey)
	t.Logf("parsed cons address: %s", addr)
}

func Test_MakeValconAddress(t *testing.T) {
	proposerAddress := "3287C60F1B7ED6F8EEC191666C13011EF7B52957"
	hrp := "ica"

	bz, _ := hex.DecodeString(proposerAddress)
	valconsAddress, _ := sdkhelper.ConvertAndEncode(hrp, bz)

	t.Logf("parsed cons address: %s", valconsAddress)
}
func Test_MakeProposerAddressFromValconsAddress(t *testing.T) {
	valconsAddress := "ica1x2ruvrcm0mt03mkpj9nxcycprmmm222hz68e5h"

	// bz, _ := hex.DecodeString(proposerAddress)
	hrp, bz, _ := sdkhelper.DecodeAndConvert(valconsAddress)

	t.Logf("parsed hrp: %s", hrp)
	t.Logf("parsed proposer bytes: %s", fmt.Sprintf("%X", bz))
}

func Test_GetConsumerValconsAddressFromPubkey(t *testing.T) {
	// input
	pubkey := "qSO+/VI0J4dRptC0EKHMC1g1lDR5t5dxGQjrufeppS4="
	hrp := "neutronvalcons"
	// output
	expectedValcons := "neutronvalcons1pgcj8m0nwuy3k6zceuswfh873w7flrnnrmdsvz"
	// test
	valconsAddr, err := sdkhelper.MakeValconsAddressFromPubeky(pubkey, hrp)
	// assert
	assert.NoError(t, err)
	assert.EqualValues(t, expectedValcons, valconsAddr)
}

func Test_CheckHexString(t *testing.T) {
	proposerAddress := "3287C60F1B7ED6F8EEC191666C13011EF7B52957"
	consensusAddress := "seivalcons1k8syjz4vtnggyjxzssmc99kcdqgyzpv3vc59pp"

	t.Log(sdkhelper.IsProposerAddress(proposerAddress))
	t.Log(sdkhelper.IsProposerAddress(consensusAddress))
}

func Test_MakeBerachainPublicKey(t *testing.T) {
	expectedPubkey := "0xa127450d48f95ad33eb34371a8e1bdbfb2eebce168c30409f303be94a93f648d0616b46688a20d8a5f87b9ad47fb6abf"
	pubkey := "oSdFDUj5WtM+s0NxqOG9v7LuvOFowwQJ8wO+lKk/ZI0GFrRmiKINil+Hua1H+2q/"
	decodedKey, _ := base64.StdEncoding.DecodeString(pubkey)
	beraPubkey := fmt.Sprintf("0x%x", decodedKey)
	assert.EqualValues(t, expectedPubkey, beraPubkey)
}

func Test_ParseBaseURL(t *testing.T) {
	rawURL := "http://example.com:8082"

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return
	}

	// Extract the hostname
	hostname := parsedURL.Hostname()

	// Print the hostname
	fmt.Println("Hostname:", hostname)
}

func Test_TimeoutCheck(t *testing.T) {
	result := waitForHeight()
	t.Log(result)
}

func waitForHeight() error {
	// Create a timeout channel using time.After
	maxWaitTime := 5 * time.Second
	timeout := time.After(maxWaitTime)
	heightFetched := make(chan bool)

	go func() {
		// Simulate fetching the latest height
		time.Sleep(3 * time.Second)
		if (time.Now().Unix() % 2) == 0 {
			heightFetched <- true // Signal that height has been fetched
		} else {
			heightFetched <- false // Still 0, continue to wait
		}
	}()

	select {
	// Check if we have exceeded the max wait time
	case <-timeout:
		return errors.New("timeout: exceeded maximum wait time")
	case success := <-heightFetched:
		if success {
			return nil // Height successfully fetched
		}
	}

	return errors.New("failed to fetch the latest height")
}

func Test_ContainFunction(t *testing.T) {
	UnsupportedChains := []string{"neutron", "stride"}
	testChainName := "neutron-testnet"
	result := helper.Contains(UnsupportedChains, testChainName)
	t.Log(result)
}
