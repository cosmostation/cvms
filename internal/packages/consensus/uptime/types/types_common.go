package types

var (
	SupportedValconsTypes  = []string{"valcons", "ica"}
	SupportedProtocolTypes = []string{"cosmos"}
)

// common
type CommonUptimeStatus struct {
	MinSignedPerWindow float64                 `json:"slash_winodw"`
	SignedBlocksWindow float64                 `json:"vote_period"`
	Validators         []ValidatorUptimeStatus `json:"validators"`
}

// cosmos uptime status
type ValidatorUptimeStatus struct {
	Moniker                   string  `json:"moniker"`
	ProposerAddress           string  `json:"proposer_address"`
	ValidatorOperatorAddress  string  `json:"validator_operator_address"`
	ValidatorConsensusAddress string  `json:"validator_consensus_addreess"`
	MissedBlockCounter        float64 `json:"missed_block_counter"`
	IsTomstoned               float64
	ProposerPriority          float64 `json:"proposer_priority"`
	// Only Consumer Chain
	ConsumerConsensusAddress string `json:"consumer_consensus_address"`
}
