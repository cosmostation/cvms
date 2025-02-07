package api

import (
	"context"

	"github.com/cosmostation/cvms/internal/common"
	"github.com/cosmostation/cvms/internal/common/api"
	"github.com/cosmostation/cvms/internal/packages/utility/upgrade/types"
)

const blockHeightInternal = 1000

func GetUpgradeStatus(
	c *common.Exporter,
	CommonUpgradeQueryPath string, CommonUpgradeParser func([]byte) (int64, string, error),
) (types.CommonUpgrade, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, common.Timeout)
	defer cancel()

	requester := c.APIClient
	resp, err := requester.Get(ctx, CommonUpgradeQueryPath)
	if err != nil {
		c.Errorf("api error: %s", err)
		return types.CommonUpgrade{}, common.ErrFailedHttpRequest
	}

	upgradeHeight, upgradeName, err := CommonUpgradeParser(resp)
	if err != nil {
		c.Errorf("parser error: %s", err)
		return types.CommonUpgrade{}, common.ErrFailedJsonUnmarshal
	}

	// non-exist onchain upgrade
	if upgradeHeight == 0 {
		c.Debugln("nothing to upgrade in on-chain state now")
		return types.CommonUpgrade{}, common.ErrCanSkip
	} else {
		c.Infof("found the onchain upgrade at %d", upgradeHeight)

		// exist onchain upgrade
		latestBlockHeight, latestBlockTimestamp, err := api.GetStatus(c.CommonClient)
		if err != nil {
			c.Errorf("api error: %s", err)
			return types.CommonUpgrade{}, common.ErrFailedHttpRequest
		}

		previousHeight, previousBlockTimestamp, _, _, _, _, err := api.GetBlock(c.CommonClient, (latestBlockHeight - blockHeightInternal))
		if err != nil {
			c.Errorf("api error: %s", err)
			return types.CommonUpgrade{}, common.ErrFailedHttpRequest
		}

		// calculate remaining time seconds
		estimatedBlockTime := (latestBlockTimestamp.Unix() - previousBlockTimestamp.Unix()) / (latestBlockHeight - previousHeight)
		remainingBlocks := upgradeHeight - latestBlockHeight
		remainingTime := remainingBlocks * estimatedBlockTime

		c.Infof("on-chain upgrade's remaining time: %d seconds", remainingTime)
		c.Infof("on-chain upgrade's remaining height: %d blocks", remainingBlocks)
		return types.CommonUpgrade{
			UpgradeName:     upgradeName,
			RemainingTime:   float64(remainingTime),
			RemainingBlocks: float64(remainingBlocks),
		}, nil
	}
}
