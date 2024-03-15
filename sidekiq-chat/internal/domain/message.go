package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CollectionMessage = "messages"
)

type Reaction struct {
	UserID       int    `json:"userId"`
	ReactionType string `json:"reactionType"`
}

type Message struct {
	Id            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GroupId       primitive.ObjectID `bson:"groupId" json:"groupId"`
	SenderID      int                `bson:"senderId" json:"senderId"`
	Message       string             `bson:"message" json:"message"`
	AttachmentUrl string             `bson:"attachment" json:"attachment"`
	Reactions     []Reaction         `bson:"reactions" json:"reactions"`
	IsDeleted     bool               `bson:"isDeleted" json:"isDeleted"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type MessageUC interface {
	Create(context.Context, Message) (Message, error)
	Delete(ctx context.Context, groupId, messageId primitive.ObjectID) error
	GetGroupMessages(ctx context.Context, groupId primitive.ObjectID, memberId int, page, pageSize int) ([]Message, error)
	GetMessageById(ctx context.Context, groupId, messageId primitive.ObjectID) (Message, error)
}

type MessageRepo interface {
	Create(context.Context, Message) (Message, error)
	Delete(ctx context.Context, groupId, messageId primitive.ObjectID) error
	GetGroupMessages(ctx context.Context, groupId primitive.ObjectID, lastViewedMessageID primitive.ObjectID, page, pageSize int) ([]Message, error)
	GetMessageById(ctx context.Context, groupId, messageId primitive.ObjectID) (Message, error)
}
