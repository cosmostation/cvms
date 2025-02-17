package repository

import (
	"context"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	idxmodel "github.com/cosmostation/cvms/internal/common/indexer/model"
	indexerrepo "github.com/cosmostation/cvms/internal/common/indexer/repository"
	dbhelper "github.com/cosmostation/cvms/internal/helper/db"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-covenant-signature/model"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

const IndexName = "babylon_covenant_signature"

type CovenantSignatureRepository struct {
	sqlTimeout time.Duration
	*bun.DB
	indexerrepo.IMetaRepository
}

func NewRepository(indexerDB common.IndexerDB, sqlTimeout time.Duration) CovenantSignatureRepository {
	// Instantiate the meta repository
	metarepo := indexerrepo.NewMetaRepository(indexerDB)

	// Return a repository that implements both IMetaRepository and vote-specific logic
	return CovenantSignatureRepository{sqlTimeout, indexerDB.DB, metarepo}
}

func (repo *CovenantSignatureRepository) InsertBabylonCovenantSignatureList(chainInfoID int64, indexPointerHeight int64, bcsList []model.BabylonCovenantSignature) error {
	ctx, cancel := context.WithTimeout(context.Background(), repo.sqlTimeout)
	defer cancel()

	// if there are not any covenant signature in this block, just update index pointer
	if len(bcsList) == 0 {
		_, err := repo.
			NewUpdate().
			Model(&idxmodel.IndexPointer{}).
			Set("pointer = ?", indexPointerHeight).
			Where("chain_info_id = ?", chainInfoID).
			Where("index_name = ?", IndexName).
			Exec(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to update new index pointer")
		}

		return nil
	}

	err := repo.RunInTx(
		ctx,
		nil,
		func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.NewInsert().
				Model(&bcsList).
				ExcludeColumn("id").
				Exec(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to insert covenant signature")
			}

			_, err = tx.
				NewUpdate().
				Model(&idxmodel.IndexPointer{}).
				Set("pointer = ?", indexPointerHeight).
				Where("chain_info_id = ?", chainInfoID).
				Where("index_name = ?", IndexName).
				Exec(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to update new index pointer")
			}

			return nil
		})

	if err != nil {
		return errors.Wrapf(err, "failed to exec covenant signature in a transaction: %v", bcsList)
	}

	return nil
}

func (repo *CovenantSignatureRepository) DeleteOldCovenantSignatureList(chainID, retentionPeriod string) (
	/* deleted rows */ int64,
	/* unexpected error */ error,
) {
	ctx, cancel := context.WithTimeout(context.Background(), repo.sqlTimeout)
	defer cancel()

	// Parsing retention period
	duration, err := dbhelper.ParseRetentionPeriod(retentionPeriod)
	if err != nil {
		return 0, err
	}

	// Calculate cutoff time duration
	cutoffTime := time.Now().Add(duration)

	// Make partition table name
	partitionTableName := dbhelper.MakePartitionTableName(IndexName, chainID)

	// Query Execution
	res, err := repo.NewDelete().
		Model((*model.BabylonCovenantSignature)(nil)).
		ModelTableExpr(partitionTableName).
		Where("timestamp < ?", cutoffTime).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := res.RowsAffected()
	return rowsAffected, nil
}
