package parser

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cosmostation/cvms/internal/common/types"
)

// celestia upgrade parser
func CelestiaUpgradeParser(resp []byte) (
	/* upgrade height */ int64,
	/* upgrade plan name  */ string,
	error) {
	var result types.CelestiaUpgradeResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, "", fmt.Errorf("parsing error: %s", err.Error())
	}

	if result.Upgrade.UpgradeHeight == "" {
		return 0, "", nil
	}

	upgradeHeight, err := strconv.ParseInt(result.Upgrade.UpgradeHeight, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("converting error: %s", err.Error())
	}
	return upgradeHeight, result.Upgrade.AppVersion, nil
}
