package types

type BabylonFinalityProviderUptimeStatues struct {
	MinSignedPerWindow      float64
	SignedBlocksWindow      float64
	FinalityProvidersStatus []FinalityProviderUptimeStatus
	LastFinalizedBlockInfo
}

type FinalityProviderUptimeStatus struct {
	Moniker            string
	Address            string
	BTCPK              string
	MissedBlockCounter float64
	Active             string
	Jailed             string
	VotingPower        float64
}

type LastFinalizedBlockInfo struct {
	MissingVotes float64
	MissingVP    float64
	FinalizedVP  float64
}
