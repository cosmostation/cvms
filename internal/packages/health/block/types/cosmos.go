package types

import "time"

const (
	CosmosBlockQueryPath    = "/block"
	CosmosBlockQueryPayload = ""
)

type CosmosBlockResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Block struct {
			Header struct {
				Height          string    `json:"height"`
				Time            time.Time `json:"time"`
				ProposerAddress string    `json:"proposer_address"`
			} `json:"header"`
		} `json:"block"`
	} `json:"result"`
}
