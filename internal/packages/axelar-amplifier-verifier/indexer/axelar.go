package indexer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/axelar-amplifier-verifier/model"
	"github.com/pkg/errors"
)

func ExtractPoll(txs []types.CosmosTx) (
	/* extracted poll in the block */ []PollVote,
	/* unexpected error */ error,
) {
	pollVoteList := make([]PollVote, 0)
	for _, tx := range txs {
		for _, message := range tx.Body.Messages {
			var preResult map[string]json.RawMessage
			err := json.Unmarshal(message, &preResult)
			if err != nil {
				return nil, err
			}

			if rawType, ok := preResult["@type"]; ok {
				var typeValue string
				err := json.Unmarshal(rawType, &typeValue)
				if err != nil {
					return nil, err
				}
				pollVotes, err := ParseDynamicMessage(message, typeValue)
				if err != nil {
					return nil, err
				}
				pollVoteList = append(pollVoteList, pollVotes...)
			}
		}
	}

	return pollVoteList, nil
}

// SUPPORTED_MESSAGE_TYPES in CVMS
const (
	AxelarAuxilaryBatchRequestTypeURL = "/axelar.auxiliary.v1beta1.BatchRequest"
	WasmExecuteContractTypeURL        = "/cosmwasm.wasm.v1.MsgExecuteContract"
)

type PollVote struct {
	VerifierAddress string
	ContractAddress string
	PollID          string
	StatusStr       string
}

func ParseDynamicMessage(message json.RawMessage, typeURL string) ([]PollVote, error) {
	switch typeURL {
	case AxelarAuxilaryBatchRequestTypeURL:
		var msg MsgBatchRequest
		if err := json.Unmarshal(message, &msg); err != nil {
			return nil, err
		}

		pollVotes := make([]PollVote, 0)
		for _, msg := range msg.Messages {
			if len(msg.Msg.Vote.Votes) == 0 {
				// log.Println(msg)
				// log.Printf("%s", message)
				// log.Println("this is not poll vote tx")
				continue
			}

			pollVotes = append(pollVotes, PollVote{
				VerifierAddress: msg.Sender,
				ContractAddress: msg.Contract,
				PollID:          msg.Msg.Vote.PollID,
				StatusStr:       msg.Msg.Vote.Votes[0],
			})
		}

		return pollVotes, nil
	default:
		return nil, nil
	}
}

type RegisterContractMsg struct {
	Type     string `json:"@type"`
	Sender   string `json:"sender"`
	Contract string `json:"contract"`
	Msg      struct {
		VerifyVerifierSet VerifyVerifierSet `json:"verify_verifier_set"`
	} `json:"msg"`
	Funds interface{} `json:"-"`
}

// VerifyVerifierSet represents the verification structure
type VerifyVerifierSet struct {
	MessageID      string         `json:"message_id"`
	NewVerifierSet NewVerifierSet `json:"new_verifier_set"`
}

// NewVerifierSet represents the new verifier set details
type NewVerifierSet struct {
	Signers   map[string]SignerDetails `json:"signers"`
	Threshold string                   `json:"threshold"`
	CreatedAt int64                    `json:"created_at"`
}

// SignerDetails holds signer-specific information
type SignerDetails struct {
	Address string `json:"address"`
	Weight  string `json:"weight"`
	PubKey  struct {
		ECDSA string `json:"ecdsa"`
	} `json:"pub_key"`
}

type MsgBatchRequest struct {
	Type     string                `json:"@type"`
	Sender   string                `json:"sender"`
	Messages []VerifierContractMsg `json:"messages"`
}

type VerifierContractMsg struct {
	Type     string `json:"@type"`
	Sender   string `json:"sender"`
	Contract string `json:"contract"`
	Msg      struct {
		Vote VoteData `json:"vote"`
	} `json:"msg"`
	Funds []interface{} `json:"-"`
}

type VoteData struct {
	PollID string   `json:"poll_id"`
	Votes  []string `json:"votes"`
}

type Poll struct {
	ContractAddress      string   `json:"_contract_address"`
	ConfirmationHeight   string   `json:"confirmation_height"`
	ExpiresAt            string   `json:"expires_at"`
	Messsage             string   `json:"message"`
	Participants         []string `json:"participants"`
	PollID               string   `json:"poll_id"`
	SourceChain          string   `json:"source_chain"`
	SourceGatewayAddress string   `json:"source_gateway_address"`
}

func (p Poll) MetaInfoString() string {
	return fmt.Sprintf("SourceChain: %s, SourceChainGatewayAddress: %s, AxelarContractAddress: %s,Message: %s",
		p.SourceChain,
		p.SourceGatewayAddress,
		p.ContractAddress,
		p.Messsage,
	)
}

const (
	pollStartType1 = "wasm-verifier_set_poll_started"
	pollStartType2 = "wasm-messages_poll_started"
)

func AmplifierPollStartFillter(events []types.BlockEvent) []Poll {
	polls := make([]Poll, 0)
	for _, event := range events {
		switch event.TypeName {
		case pollStartType1:
			poll := new(Poll)
			for _, attr := range event.Attributes {
				helper.SetFieldByTag(poll, attr.Key, attr.Value)
			}
			polls = append(polls, *poll)
		case pollStartType2:
			poll := new(Poll)
			for _, attr := range event.Attributes {
				helper.SetFieldByTag(poll, attr.Key, attr.Value)
			}
			polls = append(polls, *poll)
		}
	}
	return polls
}

// Define a struct to match JSON structure
type Contracts struct {
	VotingVerifier map[string]json.RawMessage `json:"VotingVerifier"`
}

type Contract struct {
	Address string `json:"address"`
	// etc..
}

type AxelarChainConfig struct {
	Axelar struct {
		Contracts Contracts `json:"contracts"`
	} `json:"axelar"`
}

type PollStateResponse struct {
	Data struct {
		Poll struct {
			PollID         string                 `json:"poll_id"`
			ExpfiresAt     int64                  `json:"expires_at"`
			Participantion map[string]interface{} `json:"participation"`
		} `json:"poll"`
	} `json:"data"`
}

func GetPollState(c common.CommonClient, contractAddr, pollID string) (int64, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	requester := c.APIClient.R().SetContext(ctx)
	resp, err := requester.Get(PollQueryPath(contractAddr, pollID))
	if err != nil {
		return 0, nil, errors.Errorf("rpc call is failed from %s: %s", resp.Request.URL, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return 0, nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
	}

	var result PollStateResponse
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return 0, nil, errors.Wrap(err, "failed to parse poll state")
	}

	participants := make([]string, len(result.Data.Poll.Participantion))
	for key := range result.Data.Poll.Participantion {
		participants = append(participants, key)
	}

	return result.Data.Poll.ExpfiresAt, participants, nil
}

func PollQueryPath(contractAddr, pollID string) string {
	queryMethod := fmt.Sprintf(`{"poll":{"poll_id":"%s"}}`, pollID)
	base64EncodedMethod := base64.StdEncoding.EncodeToString([]byte(queryMethod))
	return fmt.Sprintf("/cosmwasm/wasm/v1/contract/%s/smart/%s", contractAddr, base64EncodedMethod)
}

func GetVerifierContractAddressMap(c common.CommonClient, isMainnet bool) (map[string]contractInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.Timeout)
	defer cancel()

	var url string
	if isMainnet {
		url = "https://github.com/axelarnetwork/axelar-contract-deployments/raw/refs/heads/main/axelar-chains-config/info/mainnet.json"
	} else {
		url = "https://github.com/axelarnetwork/axelar-contract-deployments/raw/refs/heads/main/axelar-chains-config/info/testnet.json"
	}

	requester := c.APIClient.R().SetContext(ctx)
	resp, err := requester.Get(url)
	if err != nil {
		return nil, errors.Errorf("rpc call is failed from %s: %s", resp.Request.URL, err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, errors.Errorf("stanage status code from %s: [%d]", resp.Request.URL, resp.StatusCode())
	}

	// Extract chain names and addresses
	addressMap, err := ParseAxelarChainConfig(resp.Body())
	if err != nil {
		return nil, err
	}

	return addressMap, nil
}

// Attempt to parse object with an "address" field
type contractInfo struct {
	ChainName   string
	Address     string `json:"address"`
	BlockExpiry int    `json:"blockExpiry"`
}

// Function to extract chain names and addresses
func ParseAxelarChainConfig(resp []byte) (map[string]contractInfo, error) {
	// Parse JSON into struct
	var result AxelarChainConfig
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return nil, err
	}

	// Create a map to store chain names and addresses
	addressMap := make(map[string]contractInfo)

	// Iterate over VotingVerifier chains
	for chainName, rawJSON := range result.Axelar.Contracts.VotingVerifier {
		var contract contractInfo
		if err := json.Unmarshal(rawJSON, &contract); err == nil && contract.Address != "" {
			contract.ChainName = chainName
			addressMap[contract.Address] = contract
		} else {
			// log.Printf("Skipping non-contract entry: %s", chainName)
			continue
		}
	}

	return addressMap, nil
}

// Function to create an ID map from a list of verifiers
func CreateVerifierIDMap(verifiers []string) map[string]int64 {
	idMap := make(map[string]int64)

	// Populate the map with index as ID
	for i, verifier := range verifiers {
		idMap[verifier] = int64(i)
	}

	return idMap
}

// Convert PollMap to []AxelarAmplifierVerifierVote
func ConvertPollMapToList(pollMap PollMap) []model.AxelarAmplifierVerifierVote {
	pollVoteList := make([]model.AxelarAmplifierVerifierVote, 0)
	for _, voteList := range pollMap {
		for _, v := range voteList {
			pollVoteList = append(pollVoteList, v)
		}
	}
	return pollVoteList
}

func ConcatChainAndPollID(chainName, pollID string) string {
	return fmt.Sprintf("%s/%s", chainName, strings.ReplaceAll(pollID, `"`, ``))
}

type PollMap map[string]map[string]model.AxelarAmplifierVerifierVote
