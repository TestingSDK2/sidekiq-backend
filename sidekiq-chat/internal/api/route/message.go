package route

import (
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/api/controller"
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/gofiber/fiber/v2"
)

func RegisterMessageRoutes(app *fiber.App, gUc domain.GroupUC, groupMetaUc domain.ChatMetaUC, mUc domain.MessageUC, aUc domain.AuthUC, rUc domain.RealtimeUC) error {

	messageC := controller.NewMessageController(gUc, mUc, groupMetaUc, rUc)

	authMiddleware := AuthenticateUser(aUc)

	memberRoleMiddleware := CheckGroupMemberRole(gUc, []string{domain.RoleAdmin, domain.RoleMember})

	api := app.Group("/message")
	api.Post("/send/:groupId", authMiddleware, memberRoleMiddleware, messageC.SendMessage)
	api.Get("/list/:groupId", authMiddleware, memberRoleMiddleware, messageC.GetGroupMessages)
	api.Patch("/read/:groupId", authMiddleware, memberRoleMiddleware, messageC.UpdateReadCounter)
	api.Delete("/:groupId", authMiddleware, memberRoleMiddleware, messageC.DeleteChat)
	api.Delete("/:groupId/:messageId", authMiddleware, memberRoleMiddleware, messageC.DeleteMessage)
	return nil
}
