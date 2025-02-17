package common

import (
	"github.com/pkg/errors"

	"github.com/cosmostation/cvms/internal/helper/config"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	IndexPackages = []string{
		// consensus
		"voteindexer",
		"veindexer",
		"babylon_checkpoint",
		"babylon_covenant_signature",
		"finality-provider-indexer",
		"babylon-btc-lightclient",
		"axelar_amplifier_verifier",
	}

	ExporterPackages = []string{
		// health
		"block",
		// consensus
		"uptime",
		// utility
		"balance", "upgrade",
		// duty
		"axelar-evm", "eventnonce", "oracle", "yoda", "finality-provider-uptime",
	}
)

type Packager struct {
	Mode         Mode
	Logger       *logrus.Logger
	Factory      promauto.Factory
	ChainName    string
	Mainnet      bool
	ChainID      string
	Package      string
	ProtocolType string
	Endpoints

	// optional info by mode
	Monikers []string
	// optional info by package
	OptionPackager
}

type Endpoints struct {
	APIs     []string
	CheckAPI bool

	RPCs     []string
	CheckRPC bool

	GRPCs     []string
	CheckGRPC bool
}

type OptionPackager struct {
	// optional for balance package
	BalanceDenom     string
	BalanceExponent  int
	BalanceAddresses []string

	// optional for indexers
	*IndexerDB
	RetentionPeriod string

	// optional for consumer chain
	IsConsumerChain   bool
	ProviderEndPoints Endpoints
}

func NewPackager(
	m Mode, f promauto.Factory, l *logrus.Logger,
	mainnet bool, chainID, chainName, pkg, protocolType string,
	cc config.ChainConfig, endpoints Endpoints,
	monikers ...string, // NOTE: Some of cases, one party want to create metrics about multiple validators like moniker1, moniker2
) (*Packager, error) {
	err := packagerValidate(chainID, chainName, protocolType, endpoints)
	if err != nil {
		return nil, err
	}
	return &Packager{
		Mode:      m,
		Factory:   f,
		Logger:    l,
		Monikers:  monikers,
		Endpoints: endpoints,
		// default labels
		Mainnet:      mainnet,
		ChainID:      chainID,
		ChainName:    chainName,
		ProtocolType: protocolType,
		Package:      pkg,
	}, nil
}

func (p *Packager) SetInfoForBalancePackage(balanceAddresses []string, balanceDenom string, balanceExponent int) *Packager {
	p.BalanceAddresses = balanceAddresses
	p.BalanceDenom = balanceDenom
	p.BalanceExponent = balanceExponent
	return p
}

func (p *Packager) SetAddtionalEndpoints(providerEndpoints Endpoints) *Packager {
	p.ProviderEndPoints = providerEndpoints
	return p
}

func (p *Packager) SetConsumer() *Packager {
	p.IsConsumerChain = true
	return p
}

func (p *Packager) SetIndexerDB(idxDB *IndexerDB) *Packager {
	p.IndexerDB = idxDB
	return p
}

func packagerValidate(
	chainID, chainName, protocolType string,
	endpoints Endpoints) error {
	if chainID == "" {
		return errors.New("chain_name parameter is empty")
	}

	if chainName == "" {
		return errors.New("chain_name parameter is empty")
	}

	if protocolType == "" {
		return errors.New("chain_type parameter is empty")
	}

	if endpoints.CheckAPI && len(endpoints.APIs) == 0 {
		return errors.New("no validated API endpoints in your config")
	}

	if endpoints.CheckRPC && len(endpoints.RPCs) == 0 {
		return errors.New("no validated RPC endpoints in your config")
	}

	if endpoints.CheckGRPC && len(endpoints.GRPCs) == 0 {
		return errors.New("no validated GRPC endpoints in your config")
	}

	return nil
}
