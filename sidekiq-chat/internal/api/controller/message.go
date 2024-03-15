package controller

import (
	"strconv"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/api/response"
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type messageController struct {
	MessageUseCase   domain.MessageUC
	GroupUseCase     domain.GroupUC
	GroupMetaUseCase domain.ChatMetaUC
	RealtimeUseCase  domain.RealtimeUC
}

func NewMessageController(gu domain.GroupUC, mu domain.MessageUC, gmu domain.ChatMetaUC, rUC domain.RealtimeUC) *messageController {
	return &messageController{
		GroupUseCase:     gu,
		MessageUseCase:   mu,
		GroupMetaUseCase: gmu,
		RealtimeUseCase:  rUC,
	}
}

type SendMessageReq struct {
	Message string `json:"message"`
	// Reactions []domain.Reaction `json:"reactions"`
}

type SendMessageRes struct {
	Message        domain.Message `json:"message"`
	DeliveryStatus string         `json:"deliveryStatus"`
}

// Send Message godoc
//
//	@Summary		Send a message in group
//	@Description	This api sends message in group
//	@Tags			message
//	@Accept			json
//	@Produce		json
//
// @Param           groupId  path  string  true  "Group ID to be removed"
//
//	@Param			requestBody	body		SendMessageReq	true	"Description of the request body"
//	@Success		200	{object}	SendMessageRes
//	@Router			/message/send/{groupId} [post]
func (mc messageController) SendMessage(c *fiber.Ctx) error {
	ctx := c.Context()

	user, err := GetUserFromReq(c)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	body := new(SendMessageReq)
	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	message := domain.Message{
		GroupId:  groupId,
		Message:  body.Message,
		SenderID: user.ProfileId,
		// Reactions: body.Reactions,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		IsDeleted: false,
	}

	// store message
	inserted, err := mc.MessageUseCase.Create(ctx, message)
	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	res := SendMessageRes{
		Message: inserted,
	}

	deliverRes, err := mc.RealtimeUseCase.DeliverMessageToGroup(ctx, inserted, groupId.String(), "new_message")
	if err != nil {
		logrus.Error("Error while delivering message", err)
		res.DeliveryStatus = ""
	}
	res.DeliveryStatus = deliverRes.GetAcknowledgment().String()

	return response.SendSuccess(c, res, "")

}

type DeleteMessageRes struct {
}

// Send Message godoc
//
//	@Summary		Delete a message within group
//	@Description	Delete a message withuin group
//	@Tags			message
//	@Accept			json
//	@Produce		json
//
// @Param           groupId  path  string  true  "Group ID of the message"
// @Param           messageId  path  string  true  "Message ID of the message to be removed"
//
//	@Success		200	{object}	DeleteMessageRes
//	@Router			/message/{groupId}/{messageId} [delete]
func (mc messageController) DeleteMessage(c *fiber.Ctx) error {
	ctx := c.Context()

	user, err := GetUserFromReq(c)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// reading messageId from params and validating
	messageId, err := primitive.ObjectIDFromHex(c.Params("messageId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// checking only sender can delete own message
	message, err := mc.MessageUseCase.GetMessageById(ctx, groupId, messageId)
	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	if message.IsDeleted {
		return response.SendError(c, fiber.StatusNotFound, "message not found")
	}

	if message.SenderID != user.ProfileId {
		return response.SendError(c, fiber.StatusInternalServerError, "only owner can delete message")
	}

	err = mc.MessageUseCase.Delete(ctx, groupId, messageId)
	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, "unable to delete message")
	}

	_, err = mc.RealtimeUseCase.DeliverMessageToGroup(ctx, message, groupId.String(), "delete")
	if err != nil {
		logrus.Error("Error while delivering message", err)
	}

	return response.SendSuccess(c, nil, "message deleted successfully")

}

type GetGroupMessagesRes struct {
	Messages []domain.Message `json:"messages"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// Get messages of a group godoc
//
//	@Summary		Get messages of a group
//	@Description	This api gets all messages of a group within board.
//	@Tags			message
// @Param           groupId  path  string  true  "Group ID of the message"
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetGroupMessagesRes
//	@Router			/message/list/{groupId} [get]

func (mc messageController) GetGroupMessages(c *fiber.Ctx) error {
	ctx := c.Context()

	user, err := GetUserFromReq(c)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	page := c.Query("page", "1")
	pageSize := c.Query("pageSize", "10")

	parsedPage, err := strconv.Atoi(page)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, "unable to parse page number")
	}

	parsedPageSize, err := strconv.Atoi(pageSize)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, "unable to parse page size")
	}

	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	messages, err := mc.MessageUseCase.GetGroupMessages(ctx, groupId, user.ProfileId, parsedPage, parsedPageSize)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, "unable to get messages")
	}

	return response.SendSuccess(c, GetGroupMessagesRes{
		Messages: messages,
		Page:     parsedPage,
		PageSize: parsedPageSize,
	}, "")
}

// Delete chat for a user godoc
//
//	@Summary		delete the chat for user.
//	@Description	This api deletes the chat of a user.
//	@Tags			message
//	@Accept			json
//	@Produce		json
//
// @Param           groupId  path  string  true  "Group ID of the chat"
//
//	@Success		200	{object}	interface{}
//	@Router			/message/:groupId [delete]

func (mc messageController) DeleteChat(c *fiber.Ctx) error {
	ctx := c.Context()

	user, err := GetUserFromReq(c)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	latestMessage, err := mc.MessageUseCase.GetGroupMessages(ctx, groupId, user.ProfileId, 1, 1)
	if err != nil || len(latestMessage) == 0 {
		return response.SendError(c, fiber.StatusBadRequest, "unable to fetch last message id")
	}

	err = mc.GroupMetaUseCase.UpdateStartChatMessage(ctx, user.ProfileId, groupId, latestMessage[0].Id)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return response.SendSuccess(c, nil, "Chat delted successfully")
}

type UpdateReadCounterReq struct {
	LastSeenMessage string `json:"messageId"`
}

type UpdateReadCounterRes struct {
}

// Mark Message read godoc
//
//	@Summary		This api marks the last seen message read.
//	@Description	This api marks the last seen message read.
//	@Tags			message
//	@Accept			json
//	@Produce		json
//
// @Param           groupId  path  string  true  "Group ID of the chat"
//
//	@Success		200	{object}	UpdateReadCounterRes
//	@Router			/message/{groupId} [patch]
func (mc messageController) UpdateReadCounter(c *fiber.Ctx) error {
	ctx := c.Context()

	user, err := GetUserFromReq(c)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	body := new(UpdateReadCounterReq)
	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	messageId, err := primitive.ObjectIDFromHex(body.LastSeenMessage)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	err = mc.GroupMetaUseCase.UpdateLastSeenMessage(ctx, user.ProfileId, groupId, messageId)
	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.SendSuccess(c, nil, "Chat counter updated successfully")
}
