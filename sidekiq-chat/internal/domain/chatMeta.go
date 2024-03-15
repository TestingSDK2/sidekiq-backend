package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CollectionGroupMeta = "groupMeta"
)

type GroupMeta struct {
	Id               primitive.ObjectID `bson:"_id,omitempty"`
	GroupId          primitive.ObjectID `bson:"groupId"`
	MemberId         int                `bson:"memberId"`
	LastReadMessage  primitive.ObjectID `bson:"lastSeenMessage"`
	ChatStartMessage primitive.ObjectID `bson:"chatStartMessage"`
	CreatedAt        time.Time          `bson:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
}

type ChatMetaRepository interface {
	Create(ctx context.Context, meta GroupMeta) (GroupMeta, error)
	GetByGroupMemberId(ctx context.Context, memberId int, groupId primitive.ObjectID) (GroupMeta, error)
	UpdateStartChatMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, chatStartMessage primitive.ObjectID) error
	UpdateLastSeenMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, lastSeenMessageId primitive.ObjectID) error
}

type ChatMetaUC interface {
	Create(ctx context.Context, meta GroupMeta) (GroupMeta, error)
	GetByGroupMemberId(ctx context.Context, memberId int, groupId primitive.ObjectID) (GroupMeta, error)
	UpdateStartChatMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, chatStartMessage primitive.ObjectID) error
	UpdateLastSeenMessage(ctx context.Context, memberId int, groupId primitive.ObjectID, lastSeenMessageId primitive.ObjectID) error
}
