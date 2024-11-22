package types

import "fmt"

// ref; https://github.com/piplabs/story/blob/main/client/server/staking.go
var StoryStakingValidatorQueryPath = func(status string) string {
	return fmt.Sprintf("/staking/validators?status=%s&pagination.count_total=true&pagination.limit=500", status)
}

type StoryStakingValidatorsQueryResponse struct {
	Code int64 `json:"code"`
	Msg  struct {
		Validators []StoryStakingValidator `json:"validators"`
		Pagination struct {
			NextKey interface{} `json:"-"`
			Total   string      `json:"total"`
		} `json:"pagination"`
	}
	Error string `json:"error"`
}

type StoryStakingValidator struct {
	OperatorAddress string `json:"operator_address"`
	ConsensusPubkey struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"consensus_pubkey"`
	Description struct {
		Moniker string `json:"moniker"`
	} `json:"description"`
}

var StoryUpgradeQueryPath = "/upgrade/current_plan"

// ref; https://github.com/piplabs/story/blob/main/client/server/upgrade.go#L17
type StoryUpgradeResponse struct {
	Code int64 `json:"code"`
	Msg  struct {
		Plan struct {
			Name   string `json:"name"`
			Time   string `json:"time"`
			Height string `json:"height"`
		} `json:"plan"`
	} `json:"msg"`
	Error string `json:"error"`
}
