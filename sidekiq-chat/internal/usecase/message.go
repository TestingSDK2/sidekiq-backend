package usecase

import (
	"context"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type messageUC struct {
	messageRepository   domain.MessageRepo
	groupMetaRepository domain.ChatMetaRepository
	contextTimeout      time.Duration
}

func NewMessageUC(mr domain.MessageRepo, gmr domain.ChatMetaRepository, timeout time.Duration) domain.MessageUC {
	return messageUC{
		messageRepository:   mr,
		groupMetaRepository: gmr,
		contextTimeout:      timeout,
	}
}

func (mu messageUC) Create(ctx context.Context, message domain.Message) (domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, mu.contextTimeout)
	defer cancel()
	return mu.messageRepository.Create(ctx, message)
}

func (mu messageUC) Delete(ctx context.Context, groupId, messageId primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, mu.contextTimeout)
	defer cancel()
	return mu.messageRepository.Delete(ctx, groupId, messageId)
}

func (mu messageUC) GetGroupMessages(ctx context.Context, groupId primitive.ObjectID, memberId int, page, pageSize int) ([]domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, mu.contextTimeout)
	defer cancel()
	chatStartMessage, _ := mu.groupMetaRepository.GetByGroupMemberId(ctx, memberId, groupId)
	logrus.Info(chatStartMessage)
	return mu.messageRepository.GetGroupMessages(ctx, groupId, chatStartMessage.ChatStartMessage, page, pageSize)
}

func (mu messageUC) GetMessageById(ctx context.Context, groupId, messageId primitive.ObjectID) (domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, mu.contextTimeout)
	defer cancel()
	return mu.messageRepository.GetMessageById(ctx, groupId, messageId)
}
