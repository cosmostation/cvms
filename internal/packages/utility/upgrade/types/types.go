package types

var (
	SupportedProtocolTypes = []string{"cosmos"}
)

type CommonUpgrade struct {
	RemainingTime   float64
	RemainingHeight float64
	UpgradeName     string
}
