package grpcservice

import (
	"context"
	"encoding/json"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-notification/app"
	notificationProtobuf "github.com/TestingSDK2/sidekiq-backend/sidekiq-proto/sidekiq-notification/v1"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/sirupsen/logrus"
)

type NotificationServer struct {
	notificationProtobuf.NotificationServiceServer
	App *app.App
}

func (s *NotificationServer) MarkAllNotificationAsRead(ctx context.Context, req *notificationProtobuf.MarkAllNotificationAsReadRequest) (*notificationProtobuf.GenericReply, error) {

	err := s.App.NotificationService.MarkAllNotificationAsRead(req.ProfileID)
	if err != nil {
		logrus.Errorf("error while mark all notification as read for profileID[%s] : error[%s]", req.ProfileID, err.Error())
		return nil, err
	}

	return &notificationProtobuf.GenericReply{
		Data:    nil,
		Status:  1,
		Message: "All notifications mark as read.",
	}, nil
}

func (s *NotificationServer) MarkNotificationAsRead(ctx context.Context, req *notificationProtobuf.MarkNotificationAsReadRequest) (*notificationProtobuf.GenericReply, error) {

	err := s.App.NotificationService.MarkNotificationAsRead(req.NotificationID, req.ProfileID)
	if err != nil {
		logrus.Errorf("error while mark notification as read with ID[%s] profileID[%s] - error[%s]", req.NotificationID, req.ProfileID, err.Error())
		return nil, err
	}

	return &notificationProtobuf.GenericReply{
		Data:    nil,
		Status:  1,
		Message: "Notification mark as read.",
	}, nil
}

func (s *NotificationServer) GetNotificationList(ctx context.Context, req *notificationProtobuf.GetNotificationListRequest) (*notificationProtobuf.GenericReply, error) {

	notifications, err := s.App.NotificationService.GetNotificationList(req.ProfileID)
	if err != nil {
		logrus.Errorf("error while getting list of notification for profileID[%s] : error[%s]", req.ProfileID, err.Error())
		return nil, err
	}

	dataBytes, err := convertDataToBytes(notifications)
	if err != nil {
		logrus.Errorf("error while converting notifications to bytes: %s", err.Error())
		return nil, err
	}

	dataAny := &any.Any{
		Value: dataBytes,
	}

	return &notificationProtobuf.GenericReply{
		Data:    dataAny,
		Status:  1,
		Message: "Notification list",
	}, nil
}

func (s *NotificationServer) GetNotificationDisplayCount(ctx context.Context, req *notificationProtobuf.GetNotificationDisplayCountRequest) (*notificationProtobuf.GetNotificationDisplayCountReply, error) {

	totalcount, err := s.App.NotificationService.GetNotificationDisplayCount(req.ProfileID)
	if err != nil {
		logrus.Errorf("error while getting notification display count: %s", err.Error())
		return nil, err
	}

	return &notificationProtobuf.GetNotificationDisplayCountReply{
		Count: int32(totalcount),
	}, nil
}

func (s *NotificationServer) NotificationHandler(ctx context.Context, req *notificationProtobuf.NotificationHandlerRequest) (*notificationProtobuf.GenericReply, error) {

	err := s.App.NotificationService.NotificationHandler(req.ReceiverIDs, int(req.SenderID), req.ThingType, req.ThingID, req.ActionType, req.Message)
	if err != nil {
		logrus.Errorf("error while handling notification: %s", err.Error())
		return nil, err
	}

	return &notificationProtobuf.GenericReply{
		Data:    nil,
		Status:  1,
		Message: "Notification created and sent.",
	}, nil
}

func convertDataToBytes(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}
