package usecase

import (
	"context"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type groupUC struct {
	groupRepository domain.GroupRepository
	contextTimeout  time.Duration
}

func NewGroupUC(gr domain.GroupRepository, timeout time.Duration) domain.GroupUC {
	return groupUC{
		groupRepository: gr,
		contextTimeout:  timeout,
	}
}

func (gu groupUC) Create(ctx context.Context, group domain.Group) (domain.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.Create(ctx, group)
}

func (gu groupUC) AddMember(ctx context.Context, groupId primitive.ObjectID, member domain.GroupMember) error {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.AddMember(ctx, groupId, member)
}

func (gu groupUC) RemoveMember(ctx context.Context, groupId primitive.ObjectID, member domain.GroupMember) error {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.RemoveMember(ctx, groupId, member)
}

func (gu groupUC) UpdateMemberRole(ctx context.Context, groupId primitive.ObjectID, member domain.GroupMember) error {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.UpdateMemberRole(ctx, groupId, member)
}

func (gu groupUC) Delete(ctx context.Context, groupId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.Delete(ctx, groupId)
}

func (gu groupUC) Archive(ctx context.Context, groupId primitive.ObjectID, status bool) error {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.Archive(ctx, groupId, status)
}

func (gu groupUC) GetGroupById(ctx context.Context, groupId primitive.ObjectID) (domain.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.GetGroupById(ctx, groupId)
}

func (gu groupUC) CheckMemberInGroup(ctx context.Context, groupId primitive.ObjectID, member int) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	// get groupinfo
	groupInfo, err := gu.groupRepository.GetGroupById(ctx, groupId)
	if err != nil {
		return false, err
	}
	// check if member belongs to group, if belongs return true
	for _, memberId := range groupInfo.Members {
		if memberId.MemberId == member {
			return true, nil
		}
	}
	// if NOT member of group
	return false, nil
}

func (gu groupUC) GetGroupMemberRoleById(ctx context.Context, groupID primitive.ObjectID, memberID int) (domain.GroupMember, error) {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	// get groupinfo
	return gu.groupRepository.GetGroupMemberRoleById(ctx, groupID, memberID)

}

func (gu groupUC) GetGroupsByBoardId(ctx context.Context, boardId primitive.ObjectID, memberId int) ([]domain.Group, error) {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupRepository.GetGroupsByBoardId(ctx, boardId, memberId)
}
