package parser

import (
	"encoding/json"
	"strconv"

	"github.com/cosmostation/cvms/internal/common/types"
	"github.com/pkg/errors"
)

func ParseFinalityProviderInfos(resp []byte) (types.FinalityProviderInfosResponse, error) {
	var result types.FinalityProviderInfosResponse
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return types.FinalityProviderInfosResponse{}, errors.WithStack(err)
	}
	return result, nil
}

func ParseFinalityProviderVotings(resp []byte) (types.FinalityVotesResponse, error) {
	var result types.FinalityVotesResponse
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return types.FinalityVotesResponse{}, errors.WithStack(err)
	}

	return result, nil
}

func ParserFinalityProviderSigningInfo(resp []byte) (float64, error) {
	var result types.FinalityProviderSigningInfoResponse
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	missCounter, err := strconv.ParseFloat(result.SigningInfo.MissedBlocksCounter, 64)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return missCounter, nil
}

func ParseFinalityProviders(resp []byte) (types.FinalityProvidersResponse, error) {
	var result types.FinalityProvidersResponse
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return types.FinalityProvidersResponse{}, errors.WithStack(err)
	}

	return result, nil
}

func ParserFinalityParams(resp []byte) (float64, float64, error) {
	var result types.FinalityParams
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	signedBlocksWindow, err := strconv.ParseFloat(result.Params.SignedBlocksWindow, 64)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	minSignedPerWindow, err := strconv.ParseFloat(result.Params.MinSignedPerWindow, 64)
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}

	return signedBlocksWindow, minSignedPerWindow, nil
}

func ParserBTCLightClientParams(resp []byte) ([]string, error) {
	var result types.BabylonBTCLightClientParams
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return result.Params.InsertHeadersAllowList, nil
}

func ParserCovenantCommiteeParams(resp []byte) ([]string, error) {
	var result types.CovenantCommitteeParams
	newCovenantCommitee := make([]string, 0)

	err := json.Unmarshal(resp, &result)
	if err != nil {
		return []string{}, errors.WithStack(err)
	}

	for _, member := range result.Params.CovenantPks {
		newCovenantCommitee = append(newCovenantCommitee, member)
	}

	return newCovenantCommitee, nil
}

func ParserBTCDelegations(resp []byte) (int64, error) {
	var result types.BTCDelegationsResponse
	err := json.Unmarshal(resp, &result)
	if err != nil {
		return 0, err
	}

	delegationCount, err := strconv.ParseInt(result.Pagination.Total, 10, 64)
	if err != nil {
		return 0, err
	}
	return delegationCount, nil
}
