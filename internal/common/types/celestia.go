package types

const CelestiaUpgradeQueryPath = "/signal/v1/upgrade"

// ref; https://github.com/celestiaorg/celestia-app/blob/main/proto/celestia/signal/v1/query.proto#L22C22-L22C23
type CelestiaUpgradeResponse struct {
	Upgrade struct {
		AppVersion    string `json:"app_version"`
		UpgradeHeight string `json:"upgrade_height"`
	} `json:"upgrade"`
}
