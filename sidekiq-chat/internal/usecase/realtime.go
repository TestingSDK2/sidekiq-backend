package usecase

import (
	"context"
	"strconv"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	realtimeV1 "github.com/ProImaging/sidekiq-backend/sidekiq-proto/sidekiq-realtime/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type relatimeUC struct {
	realtimeGrpc   realtimeV1.DeliveryServiceClient
	groupRepo      domain.GroupRepository
	contextTimeout time.Duration
}

func NewRealtimeUC(realtimeGrpc realtimeV1.DeliveryServiceClient, groupRepo domain.GroupRepository, timeout time.Duration) domain.RealtimeUC {
	return relatimeUC{
		realtimeGrpc:   realtimeGrpc,
		groupRepo:      groupRepo,
		contextTimeout: timeout,
	}
}

func (rUc relatimeUC) DeliverMessageToGroup(ctx context.Context, message domain.Message, groupId string, action string) (*realtimeV1.DeliveryResponse, error) {
	res := realtimeV1.DeliveryResponse{}
	groupInfo, err := rUc.groupRepo.GetGroupById(ctx, message.GroupId)
	if err != nil {
		return &res, err
	}
	profileIds := []string{}
	for _, member := range groupInfo.Members {
		if member.MemberId == message.SenderID {
			continue
		}
		profileIds = append(profileIds, strconv.Itoa(member.MemberId))
	}

	r := &realtimeV1.MessageRequest{
		ReceiptProfileIds: profileIds,
		Message:           message.Message,
		MessageId:         message.Id.String(),
		GroupId:           message.GroupId.String(),
		Attachment:        message.AttachmentUrl,
		SenderId:          strconv.Itoa(message.SenderID),
		IsDeleted:         message.IsDeleted,
		ActionName:        action,
	}
	if action == "new_message" {
		r.Status = realtimeV1.MessageStatus_SENT
	} else if action == "delete" {
		r.Status = realtimeV1.MessageStatus_DELETED
	}
	req, err := rUc.realtimeGrpc.DeliverMessage(ctx, r)
	logrus.Error(req, err, profileIds)
	if err != nil {
		logrus.Error(err)
		return req, err
	}
	if req == nil {
		return req, errors.New("unable to deliver message")
	}
	return req, nil
}
