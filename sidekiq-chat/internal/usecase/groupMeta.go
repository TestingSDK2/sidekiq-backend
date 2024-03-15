package usecase

import (
	"context"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type groupMetaUC struct {
	groupMetaRepository domain.ChatMetaRepository
	contextTimeout      time.Duration
}

func NewGroupMetaUC(gr domain.ChatMetaRepository, timeout time.Duration) domain.ChatMetaUC {
	return groupMetaUC{
		groupMetaRepository: gr,
		contextTimeout:      timeout,
	}
}

func (gu groupMetaUC) Create(ctx context.Context, group domain.GroupMeta) (domain.GroupMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupMetaRepository.Create(ctx, group)
}

func (gu groupMetaUC) GetByGroupMemberId(ctx context.Context, memberId int, groupId primitive.ObjectID) (domain.GroupMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupMetaRepository.GetByGroupMemberId(ctx, memberId, groupId)
}

func (gu groupMetaUC) UpdateStartChatMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, chatStartMessage primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupMetaRepository.UpdateStartChatMessage(ctx, memberId, groupId, chatStartMessage)
}

func (gu groupMetaUC) UpdateLastSeenMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, lastSeenMessageId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, gu.contextTimeout)
	defer cancel()
	return gu.groupMetaRepository.UpdateLastSeenMessage(ctx, memberId, groupId, lastSeenMessageId)
}
