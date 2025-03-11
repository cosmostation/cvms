package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cosmostation/cvms/internal/common/indexer/model"

	"github.com/cosmostation/cvms/internal/helper"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

const covenantCommitteeInfoTableName = "covenant_committee_info"

func (repo *MetaRepository) CreateCovenantCommitteeInfoPartitionTableByChainID(chainID string) error {
	ctx := context.Background()
	defer ctx.Done()

	ci := &model.ChainInfo{}
	err := repo.
		NewSelect().
		Model(ci).
		Column("id").
		Where("chain_id = ?", chainID).
		Scan(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to select chain_info id by chain_id")
	}

	tableNameWithSuffix := fmt.Sprintf("meta.%s_%s", covenantCommitteeInfoTableName, helper.ParseToSchemaName(chainID))
	query := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s PARTITION OF "meta"."%s" FOR VALUES IN ('%d');`,
		tableNameWithSuffix, covenantCommitteeInfoTableName, ci.ID,
	)

	_, err = repo.NewRaw(query).Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create a new partition table")
	}

	return nil
}

func (repo *MetaRepository) GetCovenantCommitteeInfoListByChainInfoID(chainInfoID int64) ([]model.CovenantCommitteeInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	CovenantCommitteeInfoList := make([]model.CovenantCommitteeInfo, 0)
	err := repo.
		NewSelect().
		Model(&CovenantCommitteeInfoList).
		Where("chain_info_id = ?", chainInfoID).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.CovenantCommitteeInfo{}, nil
		}
		return []model.CovenantCommitteeInfo{}, errors.Wrapf(err, "failed to query covenant committee info list by chain_info_id")
	}

	return CovenantCommitteeInfoList, nil
}

func (repo *MetaRepository) InsertCovenantCommitteeInfoList(ccInfoList []model.CovenantCommitteeInfo) error {
	ctx := context.Background()
	defer ctx.Done()

	_, err := repo.NewInsert().
		Model(&ccInfoList).
		ExcludeColumn("id").
		Exec(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to insert covenant committee info list: %v", ccInfoList)
	}

	return nil
}

func (repo *MetaRepository) UpsertCovenantCommitteeInfoList(ccInfoList []model.CovenantCommitteeInfo) error {
	ctx := context.Background()
	defer ctx.Done()

	_, err := repo.NewInsert().
		Model(&ccInfoList).
		On("CONFLICT (chain_info_id, covenant_btc_pk) DO UPDATE").
		Set("moniker = EXCLUDED.moniker").
		Exec(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to upsert covenant committee info list: %v", ccInfoList)
	}

	return nil
}

func (repo *MetaRepository) GetCovenantCommitteeInfoListByMonikers(chainInfoID int64, monikers []string) ([]model.CovenantCommitteeInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	CovenantCommitteeInfoList := make([]model.CovenantCommitteeInfo, 0)
	err := repo.
		NewSelect().
		Model(&CovenantCommitteeInfoList).
		ColumnExpr("*").
		Where("chain_info_id = ?", chainInfoID).
		Where("moniker in (?)", bun.In(monikers)).
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query covenant committee info list by chain_info_id")
	}

	return CovenantCommitteeInfoList, nil
}
