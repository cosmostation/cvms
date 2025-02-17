package indexer

import (
	"log"
	"testing"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/helper/logger"
	"github.com/cosmostation/cvms/internal/packages/axelar-amplifier-verifier/model"
	"github.com/stretchr/testify/assert"
)

var (
	p = common.Packager{
		ChainName:    "axelar",
		ChainID:      "axelar-testnet-3",
		ProtocolType: "cosmos",
		Endpoints: common.Endpoints{
			RPCs: []string{"https://rpc-office.cosmostation.io/axelar-testnet"},
			APIs: []string{"https://lcd-office.cosmostation.io/axelar-testnet"},
		},
		Logger: logger.GetTestLogger(),
	}

	m = common.Packager{
		ChainName:    "axelar",
		ChainID:      "axelar-dojo-1",
		ProtocolType: "cosmos",
		Endpoints: common.Endpoints{
			RPCs: []string{"https://rpc-axelar.cosmostation.io"},
			APIs: []string{"https://g.w.lavanet.xyz/gateway/axelar/rest/3dc655f970c930f1d3e78ee71beece18"},
			// APIs: []string{"https://axelar-rest.publicnode.com"},
		},
		Logger: logger.GetTestLogger(),
	}

	// List of all verifier addresses
	verifiers = []string{
		"axelar1afj2uhx69pjclgcspfufj9dq9x87zfv0avf6we",
		"axelar1ha0xrd2ex6p4zj0962tc3dx4cm0e3m5qmuq20h",
		"axelar1awmhk4xhzh3224ydnrwxthpkz00tf7d5hw5kzk",
		"axelar1kamfmz2crw8eqcrvq8pj7pxj6l5rugvsd7cqke",
		"axelar1zxsdfexpy9lehz3fh6xjnvrpe6ze4fqzn06d55",
		"axelar12uqmh4qkax6ct0dr67c0ffurplwhrv7h5t9x42",
		"axelar13y07nxqadv3r7fq5hftz2p9rg5f9sgdpn76sf5",
		"axelar1dz6u2vy7gjyexy5zraqqqzkf4wffm75tsqn50z",
		"axelar1avayd50dt4mneu6s2s03yuwlc0swwmmzvf7f9f",
		"axelar1ejv5td70estc7ed4avnxnqqv4tpef2zafdkgms",
		"axelar15305m5uwgxt92x2k5dj4ttsx0t8kh5pmfekxth",
		"axelar1j9w5c54z5erz2awtkmztfqlues9d329x5fqps0",
		"axelar14eh260ptse8qsk80ztmeyua9qklhccyv62h9yw",
		"axelar1mcnjp5j8txr8acj2dp9v2zprwzxv3fmdy0fmsp",
		"axelar1melmdxuzk5mzs252kvykcjw2vyrqmqnke0mdyx",
		"axelar1t4dpjj6p0mgwwlxzvqqwv4w3ejs4laxz7eqxuv",
		"axelar1xt9eevxhcmrx90gc06n87em5dz4nw6v3nvxs7d",
		"axelar1aeylef34xqhrxn4mf8hpl94cya0rww9ld3ymep",
		"axelar1gc40fw08ee4vamhvtgcszladfsrd8tyhc75l3j",
		"axelar1zsq3fhmvmauev086aryquatd3jrh2mvl7wyrga",
		"axelar1l9txetl2jlne8s2h2ksv83wudvd3da3dv66fpq",
		"axelar14cteftrsu7pgx60ey4grpn66kk9def5t55asqa",
		"axelar16yf68y6g0pc64xvrn8e29mlnrarvpn74krgehp",
		"axelar1ed7zk4g6rmlph6z00p6swky65qyldxrpxw9759",
		"axelar1pcdufjvqegu5dfqr7w4ltlfjvnpf403gt5h99n",
		"axelar1mp0w0fdynzaguy909gf3ltsglnu892k555q6sm",
		"axelar1nqtlh9xmcp9d5nyl3wc77w4gwz72a8yt3kmzkf",
		"axelar13vewqf8exnav577qfdxpf60707yyazsq2hncmx",
		"axelar12umz2ds9gvtnkkmcwhukl7lm5asxjc9533dkj8",
		"axelar1j3u6kd4027wln9vnvmg449hmc3xj2m2g5uh69q",
		"axelar16dxsfhyegy40e4eqfxee5jw5gyy2xxtcw4t2na",
		"axelar1lg0d9rt4syalck9ux0hhmeeayq7njmjjdguxd6",
		"axelar1y5dkjhyeuqmkhq42wydaxvjt8j00d86t4xnjsu",
		"axelar1qtykdxw26wq9zz7pmeslqnznf0qyy3auddytn9",
		"axelar1a3qp4377kjfl3znelj6wayels428uapddpe4mp",
		"axelar1l65q24tc9e8z4dj8wj6g7t08reztazf5ur6ux2",
		"axelar1u37w5l93vx8uts5eazm8w489h9q22k026dklaq",
		"axelar1yxvh503g35quq3yacm4m8l6jurjwhqpejly30j",
		"axelar1c07q2f6v8znle7eu08hu4fvtssxxv44tu7h3fm",
		"axelar19cgyqs2welwaff6fyfyjdnnmv3lkxt9ntmg62s",
		"axelar16rfnanrns0u2cxm06ugvxej438y0gktzv9hwcl",
		"axelar19xvkln5jypz8k0x9sq66mmzawkshqxfvl9h5y8",
		"axelar1lpseq7mscuag7j9yehxmgdxh6k4ehe4hgfvfgw",
		"axelar17jjksd07c9934svqyjdkqzpmdqkjkadj5ulcpd",
		"axelar12nyeyah0j5ypfywgdd90046jgfl32tycrhlpg6",
	}
)

func Test_0_MakeLogic(t *testing.T) {
	app := common.NewCommonApp(p)
	app.SetAPIEndPoint(p.Endpoints.APIs[0])
	app.SetRPCEndPoint(p.Endpoints.RPCs[0])

	// Generate the ID map
	vidMap := CreateVerifierIDMap(verifiers)

	// 17293863 : poll start height
	// 17293864 : poll voting height
	testHeights := []int64{17293863, 17293864}

	contractAddressToChainName, err := GetVerifierContractAddressMap(app.CommonClient, false)
	assert.NoError(t, err)

	// first key: contract address
	// second key: verifier address
	pollMap := make(PollMap)

	for _, h := range testHeights {
		txsEvents, _, err := api.GetBlockResults(app.CommonClient, h)
		assert.NoError(t, err)

		polls := AmplifierPollStartFillter(txsEvents)
		if len(polls) > 0 {
			for _, poll := range polls {
				// tempList := make([]model.AxelarAmplifierVerifierVote, 0)
				log.Printf("verifiers: %d", len(poll.Participants))
				for _, verifier := range poll.Participants {
					// ✅ Ensure pollMap[contractAddress] exists before assigning a value
					chainAndPollID := ConcatChainAndPollID(poll.SourceChain, poll.PollID)
					if _, exists := pollMap[chainAndPollID]; !exists {
						pollMap[chainAndPollID] = make(map[string]model.AxelarAmplifierVerifierVote)
					}

					vote := model.AxelarAmplifierVerifierVote{
						// ID: Autoincrement
						ChainInfoID:     1,
						CreatedAt:       time.Now(),
						ChainAndPollID:  chainAndPollID,
						PollStartHeight: h,
						PollVoteHeight:  0,
						VerifierID:      vidMap[verifier],
						Status:          model.PollStart,
					}

					pollMap[chainAndPollID][verifier] = vote
					log.Printf("[%s][%s] vote: %v", chainAndPollID, verifier, vote)
				}
			}
		}

		log.Printf("inited polls: %d", len(pollMap))
		for _, innterMap := range pollMap {
			log.Printf("inited poll verifiers: %d", len(innterMap))
		}

		_, _, txs, err := api.GetBlockAndTxs(app.CommonClient, h)
		assert.NoError(t, err)

		pollVotes, err := ExtractPoll(txs)
		assert.NoError(t, err)

		log.Printf("got poll voted: %d", len(pollVotes))
		cnt := 0
		for _, pv := range pollVotes {
			contractInfo, existed := contractAddressToChainName[pv.ContractAddress]
			if !existed {
				log.Panicln("not found chain name")
			}

			key := ConcatChainAndPollID(contractInfo.ChainName, pv.PollID)
			// Ensure the outer map exists
			if _, exists := pollMap[key]; !exists {
				log.Printf("Poll key %s not found, skipping update.", key)
				continue
			}

			// Ensure the inner map contains the verifier
			item, exists := pollMap[key][pv.VerifierAddress]
			if !exists {
				log.Printf("Verifier %s not found under poll %s, skipping update.", pv.VerifierAddress, key)
				continue
			}

			// Update the status
			item.UpdateVote(pv.StatusStr, h)

			// Store the updated item back in the map
			pollMap[key][pv.VerifierAddress] = item

			cnt++
		}
		log.Printf("updated poll map: %d", cnt)
	}

	pollVoteList := ConvertPollMapToList(pollMap)
	for _, vote := range pollVoteList {
		log.Printf("vote: %v", vote)
	}

	log.Printf("got total vote list: %d", len(pollVoteList))
}

func Test_2_BatchSync(t *testing.T) {
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	p.SetIndexerDB(indexerDB)
	p.IsConsumerChain = false
	assert.NoError(t, err)

	idx, err := NewAxelarAmplifierVerifierIndexer(p)
	assert.NoError(t, err)

	err = idx.InitChainInfoID()
	assert.NoError(t, err)

	err = idx.InitPartitionTablesByChainInfoID(idx.IndexName, idx.ChainID, 100)
	assert.NoError(t, err)

	err = idx.CreateVerifierInfoPartitionTableByChainID(idx.ChainID)
	assert.NoError(t, err)

	err = idx.FetchValidatorInfoList()
	assert.NoError(t, err)

	newIndexPointer, err := idx.batchSync(17293862)
	assert.NoError(t, err)
	t.Logf("new index point: %d", newIndexPointer)
}

func Test_2_BatchSync_Mainnet(t *testing.T) {
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	m.SetIndexerDB(indexerDB)
	m.IsConsumerChain = false
	assert.NoError(t, err)

	idx, err := NewAxelarAmplifierVerifierIndexer(m)
	assert.NoError(t, err)

	err = idx.InitChainInfoID()
	assert.NoError(t, err)

	err = idx.InitPartitionTablesByChainInfoID(idx.IndexName, idx.ChainID, 1)
	assert.NoError(t, err)

	err = idx.CreateVerifierInfoPartitionTableByChainID(idx.ChainID)
	assert.NoError(t, err)

	err = idx.FetchValidatorInfoList()
	assert.NoError(t, err)

	newIndexPointer, err := idx.batchSync(16702029)
	assert.NoError(t, err)
	t.Logf("new index point: %d", newIndexPointer)
}

func Test_2_BatchSync_Mainnet2(t *testing.T) {
	tempDBName := "temp"
	indexerDB, err := common.NewTestLoaclIndexerDB(tempDBName)
	m.SetIndexerDB(indexerDB)
	m.IsConsumerChain = false
	assert.NoError(t, err)

	idx, err := NewAxelarAmplifierVerifierIndexer(m)
	assert.NoError(t, err)

	err = idx.InitChainInfoID()
	assert.NoError(t, err)

	err = idx.InitPartitionTablesByChainInfoID(idx.IndexName, idx.ChainID, 1)
	assert.NoError(t, err)

	err = idx.CreateVerifierInfoPartitionTableByChainID(idx.ChainID)
	assert.NoError(t, err)

	err = idx.FetchValidatorInfoList()
	assert.NoError(t, err)
	newIndexPointer, err := idx.batchSync(16702030)
	assert.NoError(t, err)
	t.Logf("new index point: %d", newIndexPointer)
}

func TestAmplifierPollQuery(t *testing.T) {
	app := common.NewCommonApp(m)
	app.SetAPIEndPoint(m.Endpoints.APIs[0])
	app.SetRPCEndPoint(m.Endpoints.RPCs[0])

	/*
		CONTRACT=axelar1ce9rcvw8htpwukc048z9kqmyk5zz52d5a7zqn9xlq2pg0mxul9mqxlx2cq
		QUERY='{"poll":{"poll_id":"853"}}'
		axelard query wasm contract-state smart $CONTRACT $QUERY
	*/

	contractInfo, err := GetVerifierContractAddressMap(app.CommonClient, false)
	assert.NoError(t, err)

	foundChain := "flow"
	foundPollID := "255"

	var contractAddr string
	for contract, info := range contractInfo {
		if foundChain == info.ChainName {
			contractAddr = contract
		}
	}

	expireHeight, Participants, err := GetPollState(app.CommonClient, contractAddr, foundPollID)
	assert.NoError(t, err)

	t.Logf("expire heigth: %d", expireHeight)
	t.Logf("%v", Participants)
}

func Test_0_GetContractInfo(t *testing.T) {
	app := common.NewCommonApp(m)
	app.SetAPIEndPoint(m.Endpoints.APIs[0])
	app.SetRPCEndPoint(m.Endpoints.RPCs[0])
	data, err := GetVerifierContractAddressMap(app.CommonClient, true)
	assert.NoError(t, err)
	t.Log(data)
}
