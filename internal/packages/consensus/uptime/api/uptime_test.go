package api

import (
	"testing"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/client"
	commontypes "github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/helper/logger"

	"github.com/stretchr/testify/assert"
)

func TestGetValidatorUptimeStatus(t *testing.T) {

	goodResults := make(map[string][]byte)
	goodResults["/cosmos/slashing/v1beta1/signing_infos/cosmosvalcons1m8u2gxmc92n2v6kus8u48y3u0h88kcqp69cr3a"] = []byte(`{
    "val_signing_info": {
        "address": "cosmosvalcons1m8u2gxmc92n2v6kus8u48y3u0h88kcqp69cr3a",
        "start_height": "0",
        "index_offset": "27266009",
        "jailed_until": "1970-01-01T00:00:00Z",
        "tombstoned": false,
        "missed_blocks_counter": "39"
    }
}`)

	goodResults["/cosmos/slashing/v1beta1/signing_infos/"] = []byte(`{
    "info": [
        {
            "address": "cosmosvalcons1qqqqrezrl53hujmpdch6d805ac75n220ku09rl",
            "start_height": "0",
            "index_offset": "27266009",
            "jailed_until": "1970-01-01T00:00:00Z",
            "tombstoned": false,
            "missed_blocks_counter": "1"
        },
        {
            "address": "cosmosvalcons1m8u2gxmc92n2v6kus8u48y3u0h88kcqp69cr3a",
            "start_height": "0",
            "index_offset": "27550724",
            "jailed_until": "1970-01-01T00:00:00Z",
            "tombstoned": false,
            "missed_blocks_counter": "39"
        }],
		    "pagination": {
        "total": "2"
    }
}`)

	validators := []commontypes.CosmosValidator{
		{
			Address: "D9F8A41B782AA6A66ADC81F953923C7DCE7B6001",
			Pubkey: struct {
				Type  string "json:\"type\""
				Value string "json:\"value\""
			}{
				Type:  "tendermint/PubKeyEd25519",
				Value: "CiCf1tb2Ga/BLCb8Ccm0PBk0YD7ZFKhCx+gwWxzZGtvS9g==",
			},
			VotingPower:      "1",
			ProposerPriority: "0",
		},
	}

	stakingValidators := []commontypes.CosmosStakingValidator{
		{
			OperatorAddress: "cosmosvaloper1hjct6q7npsspsg3dgvzk3sdf89spmlpfdn6m9d",
			ConsensusPubkey: commontypes.ConsensusPubkey{
				Type: "/cosmos.crypto.ed25519.PubKey",
				Key:  "CiCf1tb2Ga/BLCb8Ccm0PBk0YD7ZFKhCx+gwWxzZGtvS9g==",
			},
			Description: struct {
				Moniker string "json:\"moniker\""
			}{
				Moniker: "asdf",
			},
			Tokens: "1000",
		},
	}

	l := logger.GetTestLogger()

	commonApp := common.CommonApp{

		CommonClient: common.CommonClient{
			RPCClient:  nil,
			GRPCClient: nil,
			APIClient:  client.NewMockClient("127.0.0.1:9090", goodResults).SetLogger(l),
			Entry:      l.WithField("mode", "test"),
		},
		EndPoint:       "127.0.0.1:9090",
		OptionalClient: common.CommonClient{},
	}

	status, _ := getValidatorUptimeStatus(commonApp, "cosmoshub", validators, stakingValidators)

	assert.Equal(t, float64(39), status[0].MissedBlockCounter)
	assert.Equal(t, float64(0), status[0].IsTomstoned)

}
