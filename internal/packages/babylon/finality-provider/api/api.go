package api

import (
	"strconv"

	"github.com/cosmostation/cvms/internal/common"
	commonapi "github.com/cosmostation/cvms/internal/common/api"
	commontypes "github.com/cosmostation/cvms/internal/common/types"
	"github.com/cosmostation/cvms/internal/packages/babylon/finality-provider/types"
	"github.com/pkg/errors"
)

func GetFinalityProviderUptime(exporter *common.Exporter) (types.BabylonFinalityProviderUptimeStatues, error) {
	// 1. get finality provider infos
	finalityProviderInfos, err := commonapi.GetBabylonFinalityProviderInfos(exporter.CommonClient)
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get babylon finality provider infos")
	}
	exporter.Debugf("got total finality providers: %d", len(finalityProviderInfos))

	// 2. get latest height
	latestBlockHeight, _, err := commonapi.GetStatus(exporter.CommonClient)
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get babylon latest height")
	}

	exporter.Debugf("got latest block height: %d", latestBlockHeight)

	// 3. get active finality providers.
	activeProviders, err := commonapi.GetActiveFinalityProviderByHeight(exporter.CommonClient, latestBlockHeight)
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get babylon active finality providers")
	}

	exporter.Debugf("got active finality providers: %d", len(activeProviders))

	// 5. get lity providers' uptime status
	finalityProviderUptimeStatus, err := commonapi.GetFinalityProviderUptime(exporter.CommonClient, addActiveStatus(activeProviders, finalityProviderInfos))
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get babylon finality providers uptime")
	}
	exporter.Debugf("got active finality providers uptime status: %d", len(finalityProviderUptimeStatus))

	// 6. get on-chain uptime parameter
	signedBlocksWindow, minSignedPerWindow, err := commonapi.GetBabylonFinalityProviderParams(exporter.CommonClient)
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get babylon finality provider parameters")
	}

	// 1. check last finalized height
	lastFinalizedBlockHeight, err := commonapi.GetLastFinalizedBlockHeight(exporter.CommonClient)
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get last finalized block height")
	}

	// 2. check total votes at that height
	votes, err := commonapi.GetFinalityVotesByHeight(exporter.CommonClient, lastFinalizedBlockHeight)
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get finality votes")
	}

	// 3. check total vp at that height with only voted
	fps, err := commonapi.GetActiveFinalityProviderByHeight(exporter.CommonClient, lastFinalizedBlockHeight)
	if err != nil {
		return types.BabylonFinalityProviderUptimeStatues{}, errors.Wrap(err, "failed to get finality providers in last finalized block")
	}

	LastFinalizedBlockInfo := getLastFinalizedBlockInfo(votes, fps)

	return types.BabylonFinalityProviderUptimeStatues{
		SignedBlocksWindow:      signedBlocksWindow,
		MinSignedPerWindow:      minSignedPerWindow,
		FinalityProvidersStatus: finalityProviderUptimeStatus,
		LastFinalizedBlockInfo:  LastFinalizedBlockInfo,
	}, nil
}

func addActiveStatus(activeProviders []commontypes.FinalityProvider, finalityProviderInfos []commontypes.FinalityProviderInfo) []commontypes.FinalityProviderInfo {
	activeFpMap := make(map[string]commontypes.FinalityProvider, len(activeProviders))
	for _, fp := range activeProviders {
		activeFpMap[fp.BtcPkHex] = fp
	}

	// Modify the original slice using index-based iteration
	for i := range finalityProviderInfos {
		fp, exist := activeFpMap[finalityProviderInfos[i].BTCPK]
		if exist {
			// NOTE: fp.VotingPower must will be integer, so no need to check error
			vp, _ := strconv.ParseFloat(fp.VotingPower, 64)
			finalityProviderInfos[i].Active = true
			finalityProviderInfos[i].VotingPower = vp
		}
	}

	return finalityProviderInfos
}

func getLastFinalizedBlockInfo(votes []string, fps []commontypes.FinalityProvider) types.LastFinalizedBlockInfo {
	missingVotes := len(fps) - len(votes)
	missingVP := float64(0)
	finalizedVP := float64(0)

	finalizedVoteMap := make(map[string]bool, len(votes))
	for _, vote := range votes {
		finalizedVoteMap[vote] = true
	}

	for _, fp := range fps {
		// NOTE: fp.VotingPower must will be integer, so no need to check error
		vp, _ := strconv.ParseFloat(fp.VotingPower, 64)

		// check if fp is in votes
		exist := finalizedVoteMap[fp.BtcPkHex]
		if exist {

			finalizedVP += vp
		} else {
			missingVP += vp
		}
	}

	return types.LastFinalizedBlockInfo{
		MissingVotes: float64(missingVotes),
		MissingVP:    missingVP,
		FinalizedVP:  finalizedVP,
	}
}
