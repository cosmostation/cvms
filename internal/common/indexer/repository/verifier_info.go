package repository

import (
	"context"
	"fmt"

	"github.com/cosmostation/cvms/internal/common/indexer/model"

	"github.com/cosmostation/cvms/internal/helper"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

const verifierInfoTableName = "verifier_info"

func (repo *MetaRepository) CreateVerifierInfoPartitionTableByChainID(chainID string) error {
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

	tableNameWithSuffix := fmt.Sprintf("meta.%s_%s", verifierInfoTableName, helper.ParseToSchemaName(chainID))
	query := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s PARTITION OF "meta"."%s" FOR VALUES IN ('%d');`,
		tableNameWithSuffix, verifierInfoTableName, ci.ID,
	)

	_, err = repo.NewRaw(query).Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create a new partition table")
	}

	return nil
}

func (repo *MetaRepository) GetVerifierInfoListByChainInfoID(chainInfoID int64) ([]model.VerifierInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	verifierInfoList := make([]model.VerifierInfo, 0)
	err := repo.
		NewSelect().
		Model(&verifierInfoList).
		Where("chain_info_id = ?", chainInfoID).
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query validator_info list by chain_info_id")
	}

	return verifierInfoList, nil
}

func (repo *MetaRepository) InsertVerifierInfoList(verifierInfoList []model.VerifierInfo) error {
	ctx := context.Background()
	defer ctx.Done()

	_, err := repo.NewInsert().
		Model(&verifierInfoList).
		ExcludeColumn("id").
		Exec(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to insert validator info list")
	}

	return nil
}

func (repo *MetaRepository) GetVerifierInfoListByMonikers(chainInfoID int64, monikers []string) ([]model.VerifierInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	verifierInfoList := make([]model.VerifierInfo, 0)
	err := repo.
		NewSelect().
		Model(&verifierInfoList).
		ColumnExpr("*").
		Where("chain_info_id = ?", chainInfoID).
		Where("moniker in (?)", bun.In(monikers)).
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query validator_info list by chain_info_id")
	}

	return verifierInfoList, nil
}
