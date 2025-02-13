package repository

import (
	"context"
	"time"

	"github.com/cosmostation/cvms/internal/common"
	indexerrepo "github.com/cosmostation/cvms/internal/common/indexer/repository"
	dbhelper "github.com/cosmostation/cvms/internal/helper/db"
	"github.com/cosmostation/cvms/internal/packages/consensus/babylon-covenant-signature/model"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

const SubIndexName = "babylon_btc_delegation"

type BtcDelegationRepository struct {
	sqlTimeout time.Duration
	*bun.DB
	indexerrepo.IMetaRepository
}

func NewBtcDelegationRepository(indexerDB common.IndexerDB, sqlTimeout time.Duration) BtcDelegationRepository {
	// Instantiate the meta repository
	metarepo := indexerrepo.NewMetaRepository(indexerDB)

	// Return a repository that implements both IMetaRepository and vote-specific logic
	return BtcDelegationRepository{sqlTimeout, indexerDB.DB, metarepo}
}

func (repo *BtcDelegationRepository) InsertBabylonBtcDelegationsList(chainInfoID int64, bbdList []model.BabylonBtcDelegation) error {
	ctx, cancel := context.WithTimeout(context.Background(), repo.sqlTimeout)
	defer cancel()

	// if there are not any covenant signature in this block, just update index pointer
	if len(bbdList) == 0 {
		return nil
	}

	_, err := repo.NewInsert().
		Model(&bbdList).
		ExcludeColumn("id").
		Exec(ctx)

	if err != nil {
		return errors.Wrapf(err, "failed to insert btc delegation")
	}

	return nil
}

func (repo *BtcDelegationRepository) DeleteOldBtcDelegationList(chainID, retentionPeriod string) (
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
		Model((*model.BabylonBtcDelegation)(nil)).
		ModelTableExpr(partitionTableName).
		Where("timestamp < ?", cutoffTime).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := res.RowsAffected()
	return rowsAffected, nil
}
