package types

var (
	SupportedChains = []string{"axelar"}
)

const (
	// common
	CommonValidatorQueryPath = "/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED&pagination.count_total=true&pagination.limit=500"

	// axelar
	AxelarEvmChainsQueryPath        = "/axelar/evm/v1beta1/chains?status=1"
	AxelarChainMaintainersQueryPath = "/axelar/nexus/v1beta1/chain_maintainers/{chain}"
)

type CommonAxelarNexus struct {
	ActiveEVMChains []string
	Validators      []ValidatorStatus
}

type ValidatorStatus struct {
	Moniker                  string  `json:"moniker"`
	ValidatorOperatorAddress string  `json:"validator_operator_address"`
	Status                   float64 `json:"status"`
	EVMChainName             string  `json:"evm_chain_name"`
}
type CommonValidatorsQueryResponse struct {
	Validators []struct {
		OperatorAddress string `json:"operator_address"`
		Description     struct {
			Moniker string `json:"moniker"`
		} `json:"description"`
	} `json:"validators"`
	Pagination struct {
		NextKey interface{} `json:"-"`
		Total   string      `json:"-"`
	} `json:"-"`
}

type AxelarEvmChainsResponse struct {
	Chains []string `json:"chains"`
}

type AxelarChainMaintainersResponse struct {
	Maintainers []string `json:"maintainers"`
}
