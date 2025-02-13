package indexer

import (
	"encoding/json"
	"fmt"
	"time"
)

// SUPPORTED_MESSAGE_TYPES in CVMS
const (
	BabylonCovenantSignatureMessageType         = "/babylon.btcstaking.v1.MsgAddCovenantSigs"
	BabylonCreateBtcDelegationMessageType       = "/babylon.btcstaking.v1.MsgCreateBTCDelegation"
	BabylonCovenantSignatureReceivedMessageType = "babylon.btcstaking.v1.EventCovenantSignatureReceived"
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
