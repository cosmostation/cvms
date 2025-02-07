package common

import (
	"github.com/cosmostation/cvms/internal/helper/logger"

	"github.com/sirupsen/logrus"

	"github.com/cosmostation/cvms/internal/common/client"
)

// TODO: All Methods in VoteIndexer, we need to add here?
type IIndexer interface {
	Start() error
	Loop(lastIndexPointerHeight int64)
	FetchValidatorInfoList() error
}

// TODO
type ICollector interface {
	Start(p Packager) error
}

// Collector Function Sig
type CollectorStart func(Packager) error
type CollectorLoop func(*Exporter, Packager)

// Client
type ClientType int

const (
	RPC ClientType = iota
	API
	GRPC
)

// Methods
type Method int

const (
	GET Method = iota
	POST
)

// Application Mode

type Mode int

const (
	INVALID_APP Mode = -1   // Invalid Case
	NETWORK     Mode = iota // Network Mode to provide network status overview
	VALIDATOR               // Validator Mode to provide whole chains' status overview about validator
)

func (a Mode) String() string {
	switch {
	case a == NETWORK:
		return "Validator Monitoring System"
	case a == VALIDATOR:
		return "White List"
	default:
		return "Invalid Mode"
	}
}

type CommonClient struct {
	RPCClient  client.Client
	APIClient  client.Client
	GRPCClient client.Client
	*logrus.Entry
}
type CommonApp struct {
	CommonClient
	EndPoint string
	// optional client
	OptionalClient CommonClient
}

func NewCommonApp(p Packager) CommonApp {
	rpcClient := client.NewRestyClient().SetLogger(p.Logger)
	apiClient := client.NewRestyClient().SetLogger(p.Logger)
	grpcClient := client.NewGrpcClient(Timeout).SetLogger(p.Logger)
	entry := p.Logger.WithFields(
		logrus.Fields{
			logger.FieldKeyChain:   p.ChainName,
			logger.FieldKeyChainID: p.ChainID,
			logger.FieldKeyPackage: p.Package,
		})
	commonClient := CommonClient{rpcClient, apiClient, grpcClient, entry}
	return CommonApp{
		commonClient,
		"",
		CommonClient{},
	}
}

func (c *CommonClient) SetRPCEndPoint(endpoint string) client.Client {
	return c.RPCClient.SetEndpoint(endpoint)
}

func (c *CommonClient) GetRPCEndPoint() string {
	endpoint, _ := c.RPCClient.GetEndpoint()
	return endpoint
}

func (c *CommonClient) SetAPIEndPoint(endpoint string) client.Client {
	return c.APIClient.SetEndpoint(endpoint)
}

func (c *CommonClient) GetAPIEndPoint() string {
	endpoint, _ := c.APIClient.GetEndpoint()
	return endpoint
}

func (c *CommonClient) SetGRPCEndPoint(endpoint string) client.Client {
	return c.GRPCClient.SetEndpoint(endpoint)
}

func (c *CommonClient) GetGRPCEndPoint() string {
	endpoint, _ := c.GRPCClient.GetEndpoint()
	return endpoint
}

func NewOptionalClient(entry *logrus.Entry) CommonClient {
	rpcClient := client.NewRestyClient().SetLogger(entry.Logger)
	apiClient := client.NewRestyClient().SetLogger(entry.Logger)
	grpcClient := client.NewGrpcClient(Timeout).SetLogger(entry.Logger)
	return CommonClient{rpcClient, apiClient, grpcClient, entry}
}
