package model

import (
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

type BabylonCovenantSignature struct {
	bun.BaseModel    `bun:"table:babylon_covenant_signature"`
	ID               int64     `bun:"id,pk,autoincrement"`
	ChainInfoID      int64     `bun:"chain_info_id,pk,notnull"`
	Height           int64     `bun:"height,notnull"`
	CovenantBtcPkID  int64     `bun:"covenant_btc_pk_id,notnull"`
	BTCStakingTxHash string    `bun:"btc_staking_tx_hash,notnull"`
	Timestamp        time.Time `bun:"timestamp,notnull"`
}

func (model BabylonCovenantSignature) String() string {
	return fmt.Sprintf("BabylonCovenantSignature(ID=%d, ChainInfoID=%d, Height=%d, CovenantBtcPkID=%d, BTCStakingTxHash=%s, Timestamp=%s)",
		model.ID, model.ChainInfoID, model.Height, model.CovenantBtcPkID, model.BTCStakingTxHash, model.Timestamp.Format(time.RFC3339),
	)
}
