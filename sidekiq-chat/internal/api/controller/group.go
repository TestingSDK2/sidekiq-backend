package controller

import (
	"strings"
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/api/response"
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type groupController struct {
	GroupUseCase     domain.GroupUC
	BoardUseCase     domain.BoardUC
	GroupMetaUseCase domain.ChatMetaRepository
}

func NewGroupController(gu domain.GroupUC, bu domain.BoardUC, gmu domain.ChatMetaUC) *groupController {
	return &groupController{
		GroupUseCase:     gu,
		BoardUseCase:     bu,
		GroupMetaUseCase: gmu,
	}
}

type CreateGroupReq struct {
	Name    string               `json:"name" example:"managers" required:"true"`
	BoardId string               `json:"boardId"`
	Members []domain.GroupMember `json:"members"`
	IsGroup bool                 `json:"isGroup"`
}

type CreateGroupRes struct {
	Error error        `json:"error"`
	Group domain.Group `json:"group"`
}

// Create Group godoc
//
//	@Summary		Create a group within a board
//	@Description	This api creates a group within board with members available in group
//	@Tags			group
//	@Accept			json
//	@Produce		json
//	@Param			requestBody	body		CreateGroupReq	true	"Description of the request body"
//	@Success		200	{object}	CreateGroupRes
//	@Router			/group/create [post]
func (gc groupController) CreateGroup(c *fiber.Ctx) error {
	ctx := c.Context()

	var res CreateGroupRes

	g := new(CreateGroupReq)
	if err := c.BodyParser(g); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	user, err := GetUserFromReq(c)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// if boardId not present generate new board for members
	// if boardId present validate if all members are part of board
	// if not return error
	if g.BoardId == "" {
		boardId, err := gc.BoardUseCase.Create(ctx, g.Members, user.ProfileId, g.Name)
		if err != nil {
			return response.SendError(c, 500, err.Error())
		}
		logrus.Info("Created a new board with id:", boardId)
		g.BoardId = boardId
	} else {
		// validate board exists if not then create one with members
		// check members are in board
		members := []int{}
		for _, member := range g.Members {
			members = append(members, member.MemberId)
		}
		exists, err := gc.BoardUseCase.CheckMembersInBoard(ctx, g.BoardId, members)
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		if !exists {
			return response.SendError(c, fiber.StatusBadRequest, "board not found")
		}
	}

	// convert string board id to ObjectId
	boardId, err := primitive.ObjectIDFromHex(g.BoardId)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	for i, _ := range g.Members {
		g.Members[i].JoinedOn = time.Now()
	}

	// create entry in group collection
	group := domain.Group{Name: g.Name,
		Id:        primitive.NewObjectID(),
		Slug:      primitive.NewObjectID().Hex(),
		IsGroup:   g.IsGroup,
		BoardId:   boardId,
		Members:   g.Members,
		Owner:     user.ProfileId,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	insertedG, err := gc.GroupUseCase.Create(ctx, group)

	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	for _, member := range group.Members {
		meta, err := gc.GroupMetaUseCase.Create(ctx, domain.GroupMeta{MemberId: member.MemberId, GroupId: insertedG.Id, CreatedAt: time.Now(), UpdatedAt: time.Now()})
		logrus.Info(meta, err)
	}

	res.Group = group
	// send notification to notify service
	return response.SendSuccess(c, res, "Group created successfully")
}

type AddMemberReq struct {
	BoardId   string `json:"boardId"`
	GroupId   string `json:"groupId"`
	MemberId  int    `json:"memberId"`
	Role      string `json:"role"`
	Operation string `json:"operation"`
}

type AddMemberRes struct {
}

// Mamnge members to group godoc
//
//	@Summary		Uodate members within group of a board.
//	@Description	This api add members, remove or updates roles of member within group
//	@Tags			group
//	@Accept			json
//	@Produce		json
//
// @Param       boardId  path  string  true  "Board ID to which user belongs"
//
// @Param       groupId  path  string  true  "Group ID to be updated"
//
//	@Param			requestBody	body		AddMemberReq	true	"Description of the request body"
//	@Success		200	{object}	AddMemberRes
//	@Router			/group/update-member/{groupId} [post]
func (gc groupController) UpdateMembers(c *fiber.Ctx) error {

	ctx := c.Context()

	var res AddMemberRes

	g := new(AddMemberReq)
	if err := c.BodyParser(g); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}
	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	boardId := c.Params("boardId")

	// check if member exists in board
	exists, err := gc.BoardUseCase.CheckMembersInBoard(ctx, boardId, []int{g.MemberId})
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}
	if !exists {
		return response.SendError(c, fiber.StatusBadRequest, "board not found")
	}

	if strings.EqualFold(g.Operation, "add") {
		memberInfo, _ := gc.GroupUseCase.GetGroupMemberRoleById(ctx, groupId, g.MemberId)

		if memberInfo.MemberId != 0 {
			return response.SendError(c, fiber.StatusBadRequest, "member alreay  part of group")
		}
		// add member to group
		err = gc.GroupUseCase.AddMember(ctx, groupId, domain.GroupMember{MemberId: g.MemberId, Role: g.Role})
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, err.Error())
		}
	} else if strings.EqualFold(g.Operation, "remove") {
		// remove member to group
		err = gc.GroupUseCase.RemoveMember(ctx, groupId, domain.GroupMember{MemberId: g.MemberId, Role: g.Role})
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, err.Error())
		}
	} else if strings.EqualFold(g.Operation, "role_update") {
		// remove member to group
		err = gc.GroupUseCase.UpdateMemberRole(ctx, groupId, domain.GroupMember{MemberId: g.MemberId, Role: g.Role})
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	// notify user about operation by admin
	return response.SendSuccess(c, res, "Member updated successfully")
}

type DeleteGroup struct{}

// Delete group godoc
//
// @Summary     Delete group within board
// @Description This API deletes the group within the board.
// @Tags        group
// @Accept      json
// @Produce     json
// @Param       groupId  path  string  true  "Group ID to be removed"
// @Success     200      {object}  DeleteGroup  "Returns the response indicating success"
// @Router      /group/{groupId} [delete]
func (gc groupController) DeleteGroup(c *fiber.Ctx) error {
	ctx := c.Context()
	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	err = gc.GroupUseCase.Delete(ctx, groupId)

	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, "unable to delete group")
	}
	return response.SendSuccess(c, nil, "Group deleted successfully")

}

type ArchiveGroupReq struct {
	Archive bool `json:"archive"`
}

type ArcheiveGroupRes struct{}

// Archive group godoc
//
// @Summary     archive group within board
// @Description This API archive the group within the board.
// @Tags        group
// @Accept      json
// @Produce     json
// @Param       groupId  path  string  true  "Group ID to be archive"
// @Success     200      {object}  ArcheiveGroupRes  "Returns the response indicating success"
// @Router      /group/archive [POST]
func (gc groupController) ArchiveGroup(c *fiber.Ctx) error {
	ctx := c.Context()

	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	g := new(ArchiveGroupReq)
	if err := c.BodyParser(g); err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	err = gc.GroupUseCase.Archive(ctx, groupId, g.Archive)

	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, "unable to archive group")
	}
	msg := "Group archived successfully"
	if !g.Archive {
		msg = "Group unarchived successfully"
	}
	return response.SendSuccess(c, nil, msg)

}

// Get group godoc
//
//	@Summary		Get group info
//	@Description	This api creates a group within board with members available in group
//	@Tags			group
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string	true	"Description of the request body"
//	@Success		200	{object}	ArchiveGroupReq
//	@Failure		400	{object}	interface{}
//	@Failure		404	{object}	interface{}
//	@Failure		500	{object}	interface{}
//	@Router			/group/:groupId [get]
func (gc groupController) GetGroupById(c *fiber.Ctx) error {
	ctx := c.Context()

	// reading groupId from params and validating
	groupId, err := primitive.ObjectIDFromHex(c.Params("groupId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	group, err := gc.GroupUseCase.GetGroupById(ctx, groupId)

	if err != nil {
		return response.SendError(c, fiber.StatusInternalServerError, "group not found")
	}
	return response.SendSuccess(c, group, "")

}

type GetGroupsRes struct {
	Groups []domain.Group `json:"groups"`
}

// Get groups godoc
//
//	@Summary		Get group info present in board
//	@Description	This api get all group info within board with members available in group
//	@Tags			group
//	@Accept			json
//	@Produce		json
//	@Param			boardId	path		string	true	"Description of the request body"
//	@Success		200	{object}	GetGroupsRes
//	@Router			/group/:boardId/list [get]
func (gc groupController) GetUserGroups(c *fiber.Ctx) error {
	ctx := c.Context()
	res := GetGroupsRes{}
	user, err := GetUserFromReq(c)
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// reading boardId from params and validating
	boardId, err := primitive.ObjectIDFromHex(c.Params("boardId"))
	if err != nil {
		return response.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	groups, err := gc.GroupUseCase.GetGroupsByBoardId(ctx, boardId, user.ProfileId)
	res.Groups = groups
	if err != nil {
		return response.SendError(c, fiber.StatusNotFound, err.Error())
	}
	return response.SendSuccess(c, res, "")

}
