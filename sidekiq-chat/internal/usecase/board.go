package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/domain"
	boardGrpc "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-content/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type boardUC struct {
	boardGrpc boardGrpc.BoardServiceClient
}

func NewBoardUC(boardGrpc boardGrpc.BoardServiceClient) domain.BoardUC {
	return boardUC{
		boardGrpc: boardGrpc,
	}
}

func (bu boardUC) CheckMembersInBoard(c context.Context, boardId string, members []int) (bool, error) {
	// map to store ids of members
	membersMap := map[int]bool{}

	req := &boardGrpc.GetBoardMembersRequest{
		BoardId: boardId,
	}
	reply, err := bu.boardGrpc.GetBoardMembers(c, req)
	if err != nil {
		logrus.Error(err)
		return false, err
	}
	for _, profileID := range reply.ProfileIDs {
		logrus.Info(profileID)
		membersMap[int(profileID)] = true
	}

	for _, profileId := range members {
		if ok := membersMap[profileId]; !ok {
			return false, errors.New(fmt.Sprintf("Profile id: %d not in board", profileId))
		}
	}
	return true, nil
}

func (bu boardUC) Create(c context.Context, members []domain.GroupMember, profileId int, name string) (string, error) {
	groupMembers := []string{}
	for _, member := range members {
		groupMembers = append(groupMembers, strconv.Itoa(member.MemberId))
	}
	boardId := ""
	req := &boardGrpc.AddBoardRequest{
		ProfileID: int32(profileId),
		Board: &boardGrpc.Board{
			Title:  name,
			Admins: groupMembers,
		},
	}
	reply, err := bu.boardGrpc.AddBoard(c, req)
	if err != nil {
		logrus.Error(err)
		return boardId, err
	}
	var response struct {
		Id string `json:"_id" bson:"_id"`
	}
	err = json.Unmarshal(reply.Data.Value, &response)
	if err != nil {
		logrus.Error("Unmarshal error:", err)
		return boardId, err
	}

	return response.Id, nil
}
