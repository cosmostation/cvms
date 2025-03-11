package model

import (
	"fmt"

	"github.com/uptrace/bun"
)

type VigilanteInfo struct {
	bun.BaseModel `bun:"table:meta.vigilante_info"`

	ID              int64  `bun:"id,pk,autoincrement"`
	ChainInfoID     int64  `bun:"chain_info_id,pk,notnull"`
	OperatorAddress string `bun:"operator_address"`
	Moniker         string `bun:"moniker"`
}

func (vi VigilanteInfo) String() string {
	return fmt.Sprintf("VigilanteInfo<%d %d %s %s>",
		vi.ID,
		vi.ChainInfoID,
		vi.OperatorAddress,
		vi.Moniker,
	)
}

type VerifierInfo struct {
	bun.BaseModel `bun:"table:meta.verifier_info"`

	ID              int64  `bun:"id,pk,autoincrement"`
	ChainInfoID     int64  `bun:"chain_info_id,pk,notnull"`
	VerifierAddress string `bun:"verifier_address"`
	Moniker         string `bun:"moniker"`
}

func (vi VerifierInfo) String() string {
	return fmt.Sprintf("VerifierInfo<%d %d %s %s>",
		vi.ID,
		vi.ChainInfoID,
		vi.VerifierAddress,
		vi.Moniker,
	)
}

type CovenantCommitteeInfo struct {
	bun.BaseModel `bun:"table:meta.covenant_committee_info"`
	ID            int64  `bun:"id,pk,autoincrement"`
	ChainInfoID   int64  `bun:"chain_info_id,pk,notnull"`
	Moniker       string `bun:"moniker,notnull"`
	CovenantBtcPk string `bun:"covenant_btc_pk"`
}

func (cci CovenantCommitteeInfo) String() string {
	return fmt.Sprintf("CovenantCommitteeInfo<%d %d %s>",
		cci.ID,
		cci.ChainInfoID,
		cci.CovenantBtcPk,
	)
}

type FinalityProviderInfo struct {
	bun.BaseModel   `bun:"table:meta.finality_provider_info"`
	ID              int64  `bun:"id,pk,autoincrement"`
	ChainInfoID     int64  `bun:"chain_info_id,pk,notnull"`
	Moniker         string `bun:"moniker"`
	BTCPKs          string `bun:"btc_pk"`
	OperatorAddress string `bun:"operator_address"`
}

func (vi FinalityProviderInfo) String() string {
	return fmt.Sprintf("FinalityProviderInfo<%d %d %s %s %s>",
		vi.ID,
		vi.ChainInfoID,
		vi.BTCPKs,
		vi.OperatorAddress,
		vi.Moniker,
	)
}

type ValidatorInfo struct {
	bun.BaseModel `bun:"table:meta.validator_info"`

	ID              int64  `bun:"id,pk,autoincrement"`
	ChainInfoID     int64  `bun:"chain_info_id,pk,notnull"`
	HexAddress      string `bun:"hex_address,unique:uniq_hex_address_by_chain"`
	OperatorAddress string `bun:"operator_address,unique:uniq_operator_address_by_chain"`
	Moniker         string `bun:"moniker"`
}

func (vi ValidatorInfo) String() string {
	return fmt.Sprintf("ValidatorInfo<%d %d %s %s %s>",
		vi.ID,
		vi.ChainInfoID,
		vi.HexAddress,
		vi.OperatorAddress,
		vi.Moniker,
	)
}

type ChainInfo struct {
	bun.BaseModel `bun:"table:meta.chain_info"`

	ID        int64  `bun:"id,pk,autoincrement"`
	ChainName string `bun:"chain_name"`
	Mainnet   bool   `bun:"mainnet"`
	ChainID   string `bun:"chain_id"`
}

func (ci ChainInfo) String() string {
	return fmt.Sprintf("ChainInfo<%d %s %v %s>",
		ci.ID,
		ci.ChainName,
		ci.Mainnet,
		ci.ChainID,
	)
}

type IndexPointer struct {
	bun.BaseModel `bun:"table:meta.index_pointer"`

	ID          int64  `bun:"id,pk,autoincrement"`
	ChainInfoID int64  `bun:"chain_info_id,pk,notnull"`
	IndexName   string `bun:"index_name"`
	Pointer     int64  `bun:"pointer,notnull"`
}

func (ip IndexPointer) String() string {
	return fmt.Sprintf("IndexPointer<%d %d %s %d>",
		ip.ID,
		ip.ChainInfoID,
		ip.IndexName,
		ip.Pointer,
	)
}
