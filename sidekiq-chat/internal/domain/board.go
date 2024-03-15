package domain

import (
	"context"
)

type BoardUC interface {
	Create(c context.Context, memebers []GroupMember, profileId int, name string) (string, error)
	CheckMembersInBoard(c context.Context, boardId string, members []int) (bool, error)
}
