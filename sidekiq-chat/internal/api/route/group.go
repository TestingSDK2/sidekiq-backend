package route

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/api/controller"
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/gofiber/fiber/v2"
)

func RegisterGroupRoutes(app *fiber.App, gUc domain.GroupUC, bUc domain.BoardUC, groupMetaUseCase domain.ChatMetaUC, aUc domain.AuthUC) error {
	authMiddleware := AuthenticateUser(aUc)
	adminRoleMiddleware := CheckGroupMemberRole(gUc, []string{domain.RoleAdmin})
	memberRoleMiddleware := CheckGroupMemberRole(gUc, []string{domain.RoleAdmin, domain.RoleMember})
	isGroupMiddleware := CheckChatIsGroup(gUc)

	api := app.Group("/group")
	groupC := controller.NewGroupController(gUc, bUc, groupMetaUseCase)
	api.Post("/create", authMiddleware, groupC.CreateGroup)
	api.Post("/update-member/:boardId/:groupId", authMiddleware, adminRoleMiddleware, isGroupMiddleware, groupC.UpdateMembers)
	api.Post("/archive/:groupId/", authMiddleware, adminRoleMiddleware, isGroupMiddleware, groupC.ArchiveGroup)
	api.Get("/:groupId", authMiddleware, memberRoleMiddleware, groupC.GetGroupById)
	api.Delete("/:groupId", authMiddleware, adminRoleMiddleware, isGroupMiddleware, groupC.DeleteGroup)
	api.Get("/:boardId/list", authMiddleware, groupC.GetUserGroups)
	return nil
}
