package btcdelegation

import (
	"github.com/cosmostation/cvms/internal/common"
	commonapi "github.com/cosmostation/cvms/internal/common/api"
	"github.com/pkg/errors"
)

func GetBabylonBTCDelegationsStatus(exporter *common.Exporter) (BabylonBTCDelegationStatus, error) {
	// 1. get finality provider infos
	delegations, err := commonapi.GetBabylonBTCDelegations(exporter.CommonClient)
	if err != nil {
		return BabylonBTCDelegationStatus{}, errors.Wrap(err, "failed to get babylon btc delegations")
	}

	return delegations, nil
}
