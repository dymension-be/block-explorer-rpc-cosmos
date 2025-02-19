package backend

import (
	berpctypes "github.com/bcdevtools/block-explorer-rpc-cosmos/be_rpc/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func (m *Backend) GetStakingInfo(delegatorAddr string) (berpctypes.GenericBackendResponse, error) {
	delegatorAddr = strings.ToLower(strings.TrimSpace(delegatorAddr))
	unsafeDelegatorAddr := m.bech32Cfg.FromAnyToBech32AccountAddrUnsafe(delegatorAddr)

	resDd, err := m.queryClient.StakingQueryClient.DelegatorDelegations(m.ctx, &stakingtypes.QueryDelegatorDelegationsRequest{
		DelegatorAddr: unsafeDelegatorAddr,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to get delegator delegations").Error())
	}

	resDist, err := m.queryClient.DistributionQueryClient.DelegationTotalRewards(m.ctx, &disttypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: unsafeDelegatorAddr,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to get delegation total rewards").Error())
	}

	validatorCommission := sdk.DecCoins{}
	validatorOutstandingRewards := sdk.DecCoins{}

	isValidatorAddress := m.bech32Cfg.IsValAddr(delegatorAddr)
	if isValidatorAddress {
		resCom, err := m.queryClient.DistributionQueryClient.ValidatorCommission(m.ctx, &disttypes.QueryValidatorCommissionRequest{
			ValidatorAddress: delegatorAddr,
		})
		if err != nil {
			m.GetLogger().Error("failed to get validator commission", "error", err)
		} else {
			validatorCommission = resCom.Commission.Commission
		}

		resOutRew, err := m.queryClient.DistributionQueryClient.ValidatorOutstandingRewards(m.ctx, &disttypes.QueryValidatorOutstandingRewardsRequest{
			ValidatorAddress: delegatorAddr,
		})
		if err != nil {
			m.GetLogger().Error("failed to get validator outstanding rewards", "error", err)
		} else {
			validatorOutstandingRewards = resOutRew.Rewards.Rewards
		}
	}

	totalRewards := sdk.DecCoins{}
	for _, reward := range resDist.Rewards {
		totalRewards = totalRewards.Add(reward.Reward...)
	}

	resStakingInfo := make(berpctypes.GenericBackendResponse)
	for _, delegation := range resDd.DelegationResponses {
		resStakingInfo[delegation.Delegation.ValidatorAddress] = delegation.Balance.Amount.String()
	}

	res := berpctypes.GenericBackendResponse{
		"staking": resStakingInfo,
		"rewards": totalRewards.String(),
	}

	if !validatorCommission.IsZero() {
		res["validatorCommission"] = validatorCommission.String()
	}
	if !validatorOutstandingRewards.IsZero() {
		res["validatorOutstandingRewards"] = validatorOutstandingRewards.String()
	}

	return res, nil
}

func (m *Backend) GetValidators() (berpctypes.GenericBackendResponse, error) {
	validators, err := m.tendermintValidatorsCache.GetValidators()
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to get validators").Error())
	}

	res := make(berpctypes.GenericBackendResponse)
	for _, validator := range validators {
		consAddr := sdk.ConsAddress(validator.Address).String()
		valAddr, _, err := m.validatorsConsAddrToValAddr.GetValAddrFromConsAddr(consAddr)
		if err != nil {
			return nil, status.Error(codes.Internal, errors.Wrap(err, "failed to get valiator address from consensus address").Error())
		}
		res[consAddr] = map[string]any{
			"consAddress": consAddr,
			"valAddress":  valAddr,
			"pubKeyType":  validator.PubKey.Type(),
			"votingPower": validator.VotingPower,
		}
	}

	return res, nil
}
