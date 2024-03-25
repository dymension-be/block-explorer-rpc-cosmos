package backend

import (
	"encoding/json"
	berpctypes "github.com/bcdevtools/block-explorer-rpc-cosmos/be_rpc/types"
	berpcutils "github.com/bcdevtools/block-explorer-rpc-cosmos/be_rpc/utils"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (m *Backend) GetGovProposals(pageNo int) (berpctypes.GenericBackendResponse, error) {
	if pageNo < 1 {
		return nil, berpctypes.ErrBadPageNo
	}

	resProposals, err := m.queryClient.GovV1QueryClient.Proposals(m.ctx, &govv1types.QueryProposalsRequest{
		ProposalStatus: 0,
		Voter:          "",
		Depositor:      "",
		Pagination:     getDefaultPagination(pageNo),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	proposals := make(map[uint64]any, 0)
	for _, proposal := range resProposals.Proposals {
		proposalInfo := map[string]any{
			"id":       proposal.Id,
			"metadata": proposal.Metadata,
			"status":   proposal.Status.String(),
		}

		if len(proposal.Messages) > 0 {
			messages := make([]map[string]any, 0)
			for _, msg := range proposal.Messages {
				message := map[string]any{
					"type": msg.TypeUrl,
				}

				{
					bz, err := m.clientCtx.Codec.MarshalJSON(msg)
					if err == nil {
						msgContent := make(map[string]any)
						err = json.Unmarshal(bz, &msgContent)
						if err == nil {
							message["proto_content"] = msgContent
						}
					}
					if err != nil {
						message["proto_content_error"] = err.Error()
					}
				}

				messages = append(messages, message)
			}
			proposalInfo["messages"] = messages

		}
		if proposal.FinalTallyResult != nil {
			proposalInfo["finalTallyResult"] = map[string]string{
				"yes":        proposal.FinalTallyResult.YesCount,
				"abstain":    proposal.FinalTallyResult.AbstainCount,
				"no":         proposal.FinalTallyResult.NoCount,
				"noWithVeto": proposal.FinalTallyResult.NoWithVetoCount,
			}
		}
		if proposal.SubmitTime != nil {
			proposalInfo["submitTimeEpochUTC"] = proposal.SubmitTime.UTC().Unix()
		}
		if proposal.DepositEndTime != nil {
			proposalInfo["depositEndTimeEpochUTC"] = proposal.DepositEndTime.UTC().Unix()
		}
		if len(proposal.TotalDeposit) > 0 {
			proposalInfo["totalDeposit"] = berpcutils.CoinsToMap(proposal.TotalDeposit...)
		}
		if proposal.VotingStartTime != nil {
			proposalInfo["votingStartTimeEpochUTC"] = proposal.VotingStartTime.UTC().Unix()
		}
		if proposal.VotingEndTime != nil {
			proposalInfo["votingEndTimeEpochUTC"] = proposal.VotingEndTime.UTC().Unix()
		}

		proposals[proposal.Id] = proposalInfo
	}

	return berpctypes.GenericBackendResponse{
		"proposals": proposals,
		"pageNo":    pageNo,
		"pageSize":  defaultPageSize,
	}, nil
}
