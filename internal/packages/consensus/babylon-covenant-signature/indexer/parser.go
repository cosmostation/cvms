package indexer

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/wire"

	"bytes"
	"encoding/base64"

	"github.com/cosmostation/cvms/internal/common/types"

	"github.com/cosmostation/cvms/internal/common"
)

// sample rpc event log
// 	{
// 		"key": "covenant_btc_pk_hex",
// 		"value": "\"113c3a32a9d320b72190a04a020a0db3976ef36972673258e9a38a364f3dc3b0\"",
// 		"index": true
// 	}

// value is weird and fixed it
func DecodeEscapedJSONString(value string) (string, error) {
	var decodedString string
	err := json.Unmarshal([]byte(value), &decodedString)
	if err != nil {
		return "", err
	}

	return decodedString, nil
}

func DecodeBtcStakingTx(encodingString string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(encodingString)
	if err != nil {
		return "", err
	}

	hexStr := hex.EncodeToString(decodedBytes)

	// Convert hex string to bytes
	rawBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	// Deserialize transaction
	var tx wire.MsgTx

	err = tx.Deserialize(bytes.NewReader(rawBytes))
	if err != nil {
		return "", err
	}

	return tx.TxHash().String(), nil
}

func DecodeBTCStakingTxByHexStr(hexStr string) (string, error) {
	// Convert hex string to bytes
	rawBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	// Deserialize transaction
	var tx wire.MsgTx

	err = tx.Deserialize(bytes.NewReader(rawBytes))
	if err != nil {
		return "", err
	}

	return tx.TxHash().String(), nil
}

// {
// 	"@type": "/babylon.btcstaking.v1.MsgAddCovenantSigs",
// 	"signer": "bbn15ewur0qq3fgryz6sappwaqtqk9c45737hznule",
// 	"pk": "79a71ffd71c503ef2e2f91bccfc8fcda7946f4653cef0d9f3dde20795ef3b9f0",
// 	"staking_tx_hash": "3dce373f1142b4b30549a34adec9b9b08c6f7d01ca9fc616b2629bd244c59c43",
// 	"slashing_tx_sigs": [
// 	  "AjI88WdNZPS/eSEOOkBt1FCXKFgAcqQ1RR8bJEHCFTxI6/3vfJfTVZJz5cmOluEIUC2ShcdGL4qUc2ahPbiD/jcA"
// 	],
// 	"unbonding_tx_sig": "o7MLJCZhUtmaJ1vHMVEtzI4TNJ0OpsYgyIFXNA4cka7nnjEsJ62pcRdfzg/XuZJGmFNR+PnA72Mqsa11bnGMqQ==",
// 	"slashing_unbonding_tx_sigs": [
// 	  "AlEOxpq0GMZraZy9tfDp9Tq9KYzgWeqqVN5LTRavU+QHRkKDBCf5GZyKC3GCvnU7azKx9eVtSzz3dSFyV1mIZPIB"
// 	]
//   }

func ExtractBabylonCovenantSignature(txs []types.CosmosTx) (
	// /* block height */ int64,
	// /* block timestamp */ time.Time,
	[]MsgCovenantSignature,
	[]MsgCreateBtcDelegation,
	error,
) {
	newMsgCovenantSigs := make([]MsgCovenantSignature, 0)
	newMsgCreateBtcDelegations := make([]MsgCreateBtcDelegation, 0)

	for _, tx := range txs {
		for _, message := range tx.Body.Messages {
			var preResult map[string]json.RawMessage
			if err := json.Unmarshal(message, &preResult); err != nil {
				return nil, nil, err
			}

			if rawType, ok := preResult["@type"]; ok {
				var typeValue string
				if err := json.Unmarshal(rawType, &typeValue); err != nil {
					return nil, nil, err
				}

				parsedMsg, err := ParseDynamicMessage(message, typeValue)
				if err != nil {
					if errors.Is(err, common.ErrUnSupportedMessageType) {
						continue
					} else {
						return nil, nil, err
					}
				}

				// type check
				if msg, ok := parsedMsg.(MsgCovenantSignature); ok {
					newMsgCovenantSigs = append(newMsgCovenantSigs, msg)
				} else if msg, ok := parsedMsg.(MsgCreateBtcDelegation); ok {
					newMsgCreateBtcDelegations = append(newMsgCreateBtcDelegations, msg)
				}
			}
		}
	}

	return newMsgCovenantSigs, newMsgCreateBtcDelegations, nil
}

func ParseDynamicEvent(event types.BlockEvent) (interface{}, error) {
	switch event.TypeName {
	case BabylonCovenantSignatureReceivedEventType:
		var eventCovenantSignature EventCovenantSignature
		for _, attribute := range event.Attributes {
			switch attribute.Key {
			case "covenant_btc_pk_hex":
				eventCovenantSignature.CovenantBtcPkHex = attribute.Value
			case "covenant_unbonding_signature_hex":
				eventCovenantSignature.CovenantUnbondingSignature = attribute.Value
			case "staking_tx_hash":
				eventCovenantSignature.StakingTxHash = attribute.Value
			}
		}
		return eventCovenantSignature, nil
	case BabylonBtcDelegationCreatedEventType:
		var eventBtcDelegationCreated EventBtcDelegationCreated
		for _, attribute := range event.Attributes {
			switch attribute.Key {
			case "staking_tx_hex":
				eventBtcDelegationCreated.StakingTxHash = attribute.Value
			}
		}
		return eventBtcDelegationCreated, nil
	default:
		return nil, fmt.Errorf("%w", common.ErrUnSupportedEventType)
	}
}

// parseDynamicMessage dynamically parses the message based on its type.
func ParseDynamicMessage(message json.RawMessage, typeURL string) (interface{}, error) {
	switch typeURL {
	case BabylonCovenantSignatureMessageType:
		var msg MsgCovenantSignature
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse MsgCovenantSignature: %v", err)
			return MsgCovenantSignature{}, err
		}
		return msg, nil
	case BabylonCreateBtcDelegationMessageType:
		var msg MsgCreateBtcDelegation
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse MsgCreateBtcDelegation: %v", err)
			return MsgCreateBtcDelegation{}, err
		}
		return msg, nil
	default:
		return MsgCovenantSignature{}, fmt.Errorf("%w", common.ErrUnSupportedMessageType)
	}
}
