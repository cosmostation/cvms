package parser

import (
	"encoding/json"
	"strconv"

	"github.com/cosmostation/cvms/internal/helper"
	"github.com/cosmostation/cvms/internal/packages/health/block/types"
	"github.com/pkg/errors"
)

// cosmos
func CosmosBlockParser(resp []byte) (types.CommonBlock, error) {
	var result types.CosmosBlockResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return types.CommonBlock{}, errors.Wrap(err, "failed to unmarshal json in parser")
	}

	blockHeight, err := strconv.ParseFloat(result.Result.Block.Header.Height, 64)
	if err != nil {
		return types.CommonBlock{}, errors.Wrap(err, "failed to convert height to float")
	}

	return types.CommonBlock{
		LastBlockHeight:    blockHeight,
		LastBlockTimeStamp: float64(result.Result.Block.Header.Time.Unix()),
		ProposerAddress:    result.Result.Block.Header.ProposerAddress,
	}, nil
}

// ethereum
func EthereumBlockParser(resp []byte) (types.CommonBlock, error) {
	var result types.EthereumBlockResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return types.CommonBlock{}, errors.Wrap(err, "failed to unmarshal json in parser")
	}

	timestamp, err := helper.ParsingfromHexaNumberBaseHexaDecimal(helper.HexaNumberToInteger(result.Result.TimeStamp))
	if err != nil {
		return types.CommonBlock{}, errors.Wrap(err, "failed to convert from string to float in parser")
	}

	blockHeight, err := helper.ParsingfromHexaNumberBaseHexaDecimal(helper.HexaNumberToInteger(result.Result.Number))
	if err != nil {
		return types.CommonBlock{}, errors.Wrap(err, "failed to convert from string to float in parser")
	}

	return types.CommonBlock{
		LastBlockHeight:    float64(blockHeight),
		LastBlockTimeStamp: float64(timestamp),
	}, nil
}

// celestia
func CelestiaBlockParser(resp []byte) (types.CommonBlock, error) {
	var result types.CelestiaBlockResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return types.CommonBlock{}, errors.Wrap(err, "failed to unmarshal json in parser")
	}

	blockHeight, err := strconv.ParseFloat(result.Result.Header.Height, 64)
	if err != nil {
		return types.CommonBlock{}, errors.Wrap(err, "failed to convert from string to float in parser")
	}

	return types.CommonBlock{
		LastBlockHeight:    blockHeight,
		LastBlockTimeStamp: float64(result.Result.Header.Time.Unix()),
		ProposerAddress:    result.Result.Header.ProposerAddress,
	}, nil
}
