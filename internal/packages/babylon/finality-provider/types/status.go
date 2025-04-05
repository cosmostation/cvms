package types

type BabylonFinalityProviderUptimeStatues struct {
	MinSignedPerWindow      float64
	SignedBlocksWindow      float64
	FinalityProvidersStatus []FinalityProviderUptimeStatus
	LastFinalizedBlockInfo
	FinalityProviderTotal
}

type FinalityProviderUptimeStatus struct {
	Moniker            string
	Address            string
	BTCPK              string
	MissedBlockCounter float64
	Active             string
	Status             string
	VotingPower        float64
}

type LastFinalizedBlockInfo struct {
	MissingVotes float64
	MissingVP    float64
	FinalizedVP  float64
	BlockHeight  float64
}

type FinalityProviderTotal struct {
	Active   int
	Inactive int
	Jailed   int
	Slashed  int
}
