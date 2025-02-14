package indexer

import (
	"encoding/json"
	"fmt"
	"time"
)

// SUPPORTED_MESSAGE_TYPES in CVMS
const (
	//API message type name
	BabylonCovenantSignatureMessageType   = "/babylon.btcstaking.v1.MsgAddCovenantSigs"
	BabylonCreateBtcDelegationMessageType = "/babylon.btcstaking.v1.MsgCreateBTCDelegation"

	//RPC event type name
	BabylonCovenantSignatureReceivedEventType = "babylon.btcstaking.v1.EventCovenantSignatureReceived"
	BabylonBtcDelegationCreatedEventType      = "babylon.btcstaking.v1.EventBTCDelegationCreated"
)

var (
	// TODO: move into common api
	BlockTxsQueryPath = func(blockHeight int64) string {
		return fmt.Sprintf("/cosmos/tx/v1beta1/txs/block/%d?pagination.limit=1", blockHeight)
	}
)

type MsgCovenantSignature struct {
	Type                    string   `json:"@type"`
	Signer                  string   `json:"signer"`
	Pk                      string   `json:"pk"`
	StakingTxHash           string   `json:"staking_tx_hash"`
	SlashingTxSigs          []string `json:"slashing_tx_sigs"`
	UnbondingTxSig          string   `json:"unbonding_tx_sig"`
	SlashingUnbondingTxSigs []string `json:"slashing_unbonding_tx_sigs"`
}

// {
// 	"type": "babylon.btcstaking.v1.EventCovenantSignatureReceived",
// 	"attributes": [
// 	  {
// 		"key": "covenant_btc_pk_hex",
// 		"value": "\"17921cf156ccb4e73d428f996ed11b245313e37e27c978ac4d2cc21eca4672e4\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "covenant_unbonding_signature_hex",
// 		"value": "\"2e4eb3631fe0e89957f96d02847f9b3f84cf6137b0cc3b8b20a1f1a2b4574af43c4de3d9db3244ce2c19b98f31eb812443c6b3ec36ed5ffed7bd02c7ec1fea37\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "staking_tx_hash",
// 		"value": "\"796030db4f3a6792bb2eab671f3035ec399dce190640a189d2df29c73c1eb93a\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "msg_index",
// 		"value": "0",
// 		"index": true
// 	  }
// 	]
//   }

type EventCovenantSignature struct {
	CovenantBtcPkHex           string
	CovenantUnbondingSignature string
	StakingTxHash              string
}

// {
// 	"type": "babylon.btcstaking.v1.EventBTCDelegationCreated",
// 	"attributes": [
// 	  {
// 		"key": "finality_provider_btc_pks_hex",
// 		"value": "[\"e4889630fa8695dae630c41cd9b85ef165ccc2dc5e5935d5a24393a9defee9ef\"]",
// 		"index": true
// 	  },
// 	  {
// 		"key": "new_state",
// 		"value": "\"PENDING\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "params_version",
// 		"value": "\"5\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "staker_btc_pk_hex",
// 		"value": "\"7e98b3ff96932e5555f7b248385440973ac0907316beb6357cd0197a8d33f7bf\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "staking_output_index",
// 		"value": "\"0\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "staking_time",
// 		"value": "\"64000\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "staking_tx_hex",
// 		"value": "\"0200000002500971b46b848e8e90be6504b46bc28995109a8780dcb2ee6f3058562ca77dad0200000000fffffffffd23c7e6a90c58618aa61be69ad51ddba33e7b07172000307aa75b3b969149bd0000000000ffffffff0250c3000000000000225120088e73ac52083d059125d16c1e7b6d90f3fcaf5c3df8d6bdddba747a6f3c86ef7f960000000000002251207049b01dae5d04b70ec3d92cb23c1122aca4a135f80a176d3f077a62510005cc00000000\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "unbonding_time",
// 		"value": "\"1008\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "unbonding_tx",
// 		"value": "\"020000000135e5aa77290337549f296bb6e214cbac009e07595c86ad34c00d03d3a50145e80000000000ffffffff0180bb0000000000002251204826695fd00765abb95fa58d11633fd689e0e9540947c411b8894c36cdcd51c100000000\"",
// 		"index": true
// 	  },
// 	  {
// 		"key": "msg_index",
// 		"value": "0",
// 		"index": true
// 	  }
// 	]
//   }

type EventBtcDelegationCreated struct {
	StakingTxHash string
}

type MsgCreateBtcDelegation struct {
	Type       string `json:"@type"`
	StakerAddr string `json:"staker_addr"`
	Pop        struct {
		BtcSigType string `json:"btc_sig_type"`
		BtcSig     string `json:"btc_sig"`
	} `json:"pop"`
	BtcPk                         string   `json:"btc_pk"`
	FpBtcPkList                   []string `json:"fp_btc_pk_list"`
	StakingTime                   int      `json:"staking_time"`
	StakingValue                  string   `json:"staking_value"`
	StakingTx                     string   `json:"staking_tx"`
	StakingTxInclusionProof       any      `json:"staking_tx_inclusion_proof"`
	SlashingTx                    string   `json:"slashing_tx"`
	DelegatorSlashingSig          string   `json:"delegator_slashing_sig"`
	UnbondingTime                 int      `json:"unbonding_time"`
	UnbondingTx                   string   `json:"unbonding_tx"`
	UnbondingValue                string   `json:"unbonding_value"`
	UnbondingSlashingTx           string   `json:"unbonding_slashing_tx"`
	DelegatorUnbondingSlashingSig string   `json:"delegator_unbonding_slashing_sig"`
}

type CovenantSignature struct {
	BlockHeight                int64
	CovenantPk                 string
	BTCStakingTxHash           string
	CovenantUnbondingSignature string
	Timestamp                  time.Time
}

// TODO: I think this types should move into common cosmos types
type CosmosTx struct {
	Body struct {
		Messages []json.RawMessage `json:"messages"`
	} `json:"body"`
	AuthInfo   interface{} `json:"-"`
	Signatures []string    `json:"-"`
}

type BlockTxsResponse struct {
	Txs   []CosmosTx `json:"txs"`
	Block struct {
		Header struct {
			ChainID         string    `json:"chain_id"`
			Height          string    `json:"height"`
			Time            time.Time `json:"time"`
			ProposerAddress string    `json:"proposer_address"`
		} `json:"header"`
	} `json:"block"`
}
