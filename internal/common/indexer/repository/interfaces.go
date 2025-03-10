package repository

import "github.com/cosmostation/cvms/internal/common/indexer/model"

// common repo interface
type IMetaRepository interface {
	IChainInfoRepository
	IIndexPointerRepository
	IValidatorInfoRepository
	IFinalityProviderInfoRepository
	ICovenantCommitteeInfoRepository
	IVerifierInfoRepository
	IVigilanteInfoRepository

	// common sql interface for partition tables
	CreatePartitionTable(IndexName, chainID string) error
	InitPartitionTablesByChainInfoID(IndexName, chainID string, latestHeight int64) error
}

// interface for about meta.chain_info table
type IChainInfoRepository interface {
	InsertChainInfo(chainName, chainID string, IsMainnet bool) (int64, error)
	SelectChainInfoIDByChainID(chainID string) (int64, error)
}

// interface for about meta.index_pointer table
type IIndexPointerRepository interface {
	InitializeIndexPointerByChainID(indexTableName, chainID string, startHeight int64) error
	GetLastIndexPointerByIndexTableName(indexTableName string, chainInfoID int64) (model.IndexPointer, error)
	CheckIndexPointerAlreadyInitialized(indexTableName string, chainInfoID int64) (bool, error)
}

// interface for about meta.validator_info table
type IValidatorInfoRepository interface {
	CreateValidatorInfoPartitionTableByChainID(chainID string) error
	GetValidatorInfoListByChainInfoID(chainInfoID int64) (validatorInfoList []model.ValidatorInfo, err error)
	InsertValidatorInfoList(validatorInfoList []model.ValidatorInfo) error
	GetValidatorInfoListByMonikers(chainInfoID int64, monikers []string) ([]model.ValidatorInfo, error)
}

// interface for about meta.finality_provider table
type IFinalityProviderInfoRepository interface {
	CreateFinalityProviderInfoPartitionTableByChainID(chainID string) error
	GetFinalityProviderInfoListByChainInfoID(chainInfoID int64) (fpInfoList []model.FinalityProviderInfo, err error)
	InsertFinalityProviderInfoList([]model.FinalityProviderInfo) error
	GetFinalityProviderInfoListByMonikers(chainInfoID int64, monikers []string) ([]model.FinalityProviderInfo, error)
}

// interface for about meta.covenant_committee table
type ICovenantCommitteeInfoRepository interface {
	CreateCovenantCommitteeInfoPartitionTableByChainID(chainID string) error
	GetCovenantCommitteeInfoListByChainInfoID(chainInfoID int64) (fpInfoList []model.CovenantCommitteeInfo, err error)
	UpsertCovenantCommitteeInfoList([]model.CovenantCommitteeInfo) error
	GetCovenantCommitteeInfoListByMonikers(chainInfoID int64, monikers []string) ([]model.CovenantCommitteeInfo, error)
}

// interface for about meta.verifier_info table
type IVerifierInfoRepository interface {
	CreateVerifierInfoPartitionTableByChainID(chainID string) error
	GetVerifierInfoListByChainInfoID(chainInfoID int64) (verifierInfoList []model.VerifierInfo, err error)
	InsertVerifierInfoList(verifierInfoList []model.VerifierInfo) error
	GetVerifierInfoListByMonikers(chainInfoID int64, monikers []string) ([]model.VerifierInfo, error)
}

// interface for about meta.verifier_info table
type IVigilanteInfoRepository interface {
	CreateVigilanteInfoPartitionTableByChainID(chainID string) error
	GetVigilanteInfoListByChainInfoID(chainInfoID int64) (verifierInfoList []model.VigilanteInfo, err error)
	InsertVigilanteInfoList(verifierInfoList []model.VigilanteInfo) error
	GetVigilanteInfoListByMonikers(chainInfoID int64, monikers []string) ([]model.VigilanteInfo, error)
}
