package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	CollectionGroup = "groups"
	RoleAdmin       = "admin"
	RoleMember      = "member"
)

type GroupMember struct {
	MemberId int       `bson:"memberId" json:"memberId"`
	Role     string    `bson:"role" json:"role"`
	JoinedOn time.Time `bson:"joinedOn" json:"joinedOn"`
}

type Group struct {
	Id            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Slug          string             `bson:"slug" validate:"required,unique" json:"slug"`
	Name          string             `bson:"name" json:"name"`
	Members       []GroupMember      `bson:"members" json:"members"`
	FormerMembers []GroupMember      `bson:"formerMembers" json:"formerMembers"`
	IsArchive     bool               `bson:"isArchive" json:"isArchive"`
	IsDeleted     bool               `bson:"isDeleted" json:"isDeleted"`
	IsGroup       bool               `bson:"isGroup" json:"isGroup"`
	BoardId       primitive.ObjectID `bson:"boardId" json:"boardId"`
	Owner         int                `bson:"owner" json:"owner"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type GroupRepository interface {
	GetGroupById(ctx context.Context, id primitive.ObjectID) (Group, error)
	Create(context.Context, Group) (Group, error)
	AddMember(ctx context.Context, groupID primitive.ObjectID, member GroupMember) error
	RemoveMember(ctx context.Context, groupID primitive.ObjectID, member GroupMember) error
	UpdateMemberRole(ctx context.Context, groupID primitive.ObjectID, member GroupMember) error
	Delete(ctx context.Context, groupID primitive.ObjectID) error
	Archive(ctx context.Context, groupID primitive.ObjectID, status bool) error
	GetGroupMemberRoleById(ctx context.Context, groupID primitive.ObjectID, memberID int) (GroupMember, error)
	GetGroupsByBoardId(ctx context.Context, boardId primitive.ObjectID, memberId int) ([]Group, error)
}

type GroupUC interface {
	GetGroupById(ctx context.Context, id primitive.ObjectID) (Group, error)
	Create(c context.Context, g Group) (Group, error)
	AddMember(ctx context.Context, groupID primitive.ObjectID, member GroupMember) error
	RemoveMember(ctx context.Context, groupID primitive.ObjectID, member GroupMember) error
	UpdateMemberRole(ctx context.Context, groupID primitive.ObjectID, member GroupMember) error
	Delete(ctx context.Context, groupID primitive.ObjectID) error
	Archive(ctx context.Context, groupID primitive.ObjectID, status bool) error
	CheckMemberInGroup(ctx context.Context, groupId primitive.ObjectID, member int) (bool, error)
	GetGroupMemberRoleById(ctx context.Context, groupID primitive.ObjectID, memberID int) (GroupMember, error)
	GetGroupsByBoardId(ctx context.Context, boardId primitive.ObjectID, memberId int) ([]Group, error)
}
