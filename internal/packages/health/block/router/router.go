package router

import (
	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/packages/health/block/api"
	"github.com/cosmostation/cvms/internal/packages/health/block/parser"
	"github.com/cosmostation/cvms/internal/packages/health/block/types"
)

func GetStatus(client *common.Exporter, protocolType string) (types.CommonBlock, error) {
	var (
		CommonBlockCallClient common.ClientType
		CommonBlockCallMethod common.Method
		CommonBlockQueryPath  string
		CommonBlockPayload    string
		CommonBlockParser     types.BlockParser
	)

	switch protocolType {
	case "cosmos":
		CommonBlockCallClient = common.RPC
		CommonBlockCallMethod = common.GET
		CommonBlockQueryPath = types.CosmosBlockQueryPath
		CommonBlockPayload = types.CosmosBlockQueryPayload
		CommonBlockParser = parser.CosmosBlockParser

		return api.GetBlockStatus(client, CommonBlockCallClient, CommonBlockCallMethod, CommonBlockQueryPath, CommonBlockPayload, CommonBlockParser)

	case "ethereum":
		CommonBlockCallClient = common.RPC
		CommonBlockCallMethod = common.POST
		CommonBlockQueryPath = types.EthereumBlockQueryPath
		CommonBlockPayload = types.EthereumBlockQueryPayLoad
		CommonBlockParser = parser.EthereumBlockParser

		return api.GetBlockStatus(client, CommonBlockCallClient, CommonBlockCallMethod, CommonBlockQueryPath, CommonBlockPayload, CommonBlockParser)

	case "celestia":
		CommonBlockCallClient = common.RPC
		CommonBlockCallMethod = common.POST
		CommonBlockQueryPath = types.CelestiaBlockQueryPath
		CommonBlockPayload = types.CelestiaBlockQueryPayLoad
		CommonBlockParser = parser.CelestiaBlockParser

		return api.GetBlockStatus(client, CommonBlockCallClient, CommonBlockCallMethod, CommonBlockQueryPath, CommonBlockPayload, CommonBlockParser)

	default:
		return types.CommonBlock{}, common.ErrOutOfSwitchCases
	}
}
