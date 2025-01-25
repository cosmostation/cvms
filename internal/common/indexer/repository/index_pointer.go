package repository

import (
	"context"
	"database/sql"

	"github.com/cosmostation/cvms/internal/common/indexer/model"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

func (repo *MetaRepository) InitializeIndexPointerByChainID(
	indexTableName, chainID string, startHeight int64, // for init-latest flag
) error {
	ctx := context.Background()
	defer ctx.Done()

	// phase 1
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

	// phase 2
	exists, err := repo.
		NewSelect().
		Model((*model.IndexPointer)(nil)).
		Where("chain_info_id = ?", ci.ID).
		Where("index_name = ?", indexTableName).
		Exists(ctx)
	if err != nil {
		// already initialized index pointer for current chain_info_id
		return errors.Wrap(err, "failed to check index_pointer was already exsited in the meta.index_point")
	}

	// phase 3
	if !exists {
		err := repo.RunInTx(
			ctx,
			nil,
			func(ctx context.Context, tx bun.Tx) error {
				initPointer := startHeight
				_, err = tx.
					NewInsert().
					Model(&model.IndexPointer{
						ChainInfoID: ci.ID,
						IndexName:   indexTableName,
						Pointer:     initPointer,
					}).
					On("CONFLICT ON CONSTRAINT uniq_index_name_by_chain_info_id DO NOTHING").
					Exec(ctx)
				if err != nil {
					return errors.Wrapf(err, "failed to init index pointer")
				}
				return nil
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func (repo *MetaRepository) GetLastIndexPointerByIndexTableName(indexTableName string, chainInfoID int64) (model.IndexPointer, error) {
	ctx := context.Background()
	defer ctx.Done()

	var ip model.IndexPointer
	err := repo.
		NewSelect().
		Model(&ip).
		Where("chain_info_id = ?", chainInfoID).
		Where("index_name = ?", indexTableName).
		Scan(ctx)
	if err != nil {
		return ip, errors.Wrapf(err, "failed to query validator_info list by chain_info_id: %s/%d", indexTableName, chainInfoID)
	}

	return ip, nil
}

func (repo *MetaRepository) CheckIndexpoinerAlreadyInitialized(indexTableName string, chainInfoID int64) (bool, error) {
	ctx := context.Background()
	defer ctx.Done()

	var ip model.IndexPointer
	err := repo.
		NewSelect().
		Model(&ip).
		Where("chain_info_id = ?", chainInfoID).
		Where("index_name = ?", indexTableName).
		Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to select count(*) in index_pointer table where validator_miss")
	}
	return true, nil
}
