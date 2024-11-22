package router

import (
	"github.com/cosmostation/cvms/internal/common"
	commonparser "github.com/cosmostation/cvms/internal/common/parser"
	commontypes "github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/packages/utility/upgrade/api"
	"github.com/cosmostation/cvms/internal/packages/utility/upgrade/types"
)

func GetStatus(client *common.Exporter, chainName string) (types.CommonUpgrade, error) {
	var (
		commonUpgradeQueryPath string
		commonUpgradeParser    func([]byte) (int64, string, error)
	)

	switch chainName {
	case "celestia":
		commonUpgradeQueryPath = commontypes.CelestiaUpgradeQueryPath
		commonUpgradeParser = commonparser.CelestiaUpgradeParser
		return api.GetUpgradeStatus(client, commonUpgradeQueryPath, commonUpgradeParser)

	case "story":
		commonUpgradeQueryPath = commontypes.StoryUpgradeQueryPath
		commonUpgradeParser = commonparser.StoryUpgradeParser
		return api.GetUpgradeStatus(client, commonUpgradeQueryPath, commonUpgradeParser)

	default:
		commonUpgradeQueryPath = commontypes.CosmosUpgradeQueryPath
		commonUpgradeParser = commonparser.CosmosUpgradeParser
		return api.GetUpgradeStatus(client, commonUpgradeQueryPath, commonUpgradeParser)
	}
}
