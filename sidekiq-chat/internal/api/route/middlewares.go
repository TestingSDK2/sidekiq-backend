package route

import (
	"strconv"
	"strings"

	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/api/controller"
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/api/response"
	"github.com/ProImaging/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func AuthenticateUser(aUc domain.AuthUC) fiber.Handler {
	// This function returns the middleware handler
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		token := c.Get("rs-sidkiq-auth-token")
		if token == "" {
			cookieToken := c.Cookies("rs-sidkiq-auth-token")
			if cookieToken == "" {
				return response.SendError(c, fiber.StatusUnauthorized, "token invalid")
			}
			token = cookieToken
		}
		profileId := c.Get("Profile")
		parsedProfileId, err := strconv.Atoi(profileId)
		if err != nil {
			return response.SendError(c, fiber.StatusUnauthorized, "Invalid profile")
		}
		user, err := aUc.ValidateUser(ctx, token, int32(parsedProfileId), true)
		if err != nil {
			logrus.Error(err)
			return response.SendError(c, fiber.StatusUnauthorized, "invalid token")

		}
		userInfo := domain.User{
			Id:        int(user.Data.Id),
			ProfileId: parsedProfileId,
		}
		c.Locals("userInfo", userInfo)
		return c.Next()
	}
}

func CheckGroupMemberRole(gUc domain.GroupUC, roles []string) fiber.Handler {
	// This function returns the middleware handler
	return func(c *fiber.Ctx) error {
		user, err := controller.GetUserFromReq(c)
		if err != nil {
			return response.SendError(c, fiber.StatusForbidden, "unable to fetch user")
		}
		groupID, err := primitive.ObjectIDFromHex(c.Params("groupID"))
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		// Check if the member belongs to the group
		member, err := gUc.GetGroupMemberRoleById(c.Context(), groupID, user.ProfileId)
		if err != nil {
			return response.SendError(c, fiber.StatusForbidden, "member not in group")
		}
		isAuth := false
		for _, role := range roles {
			if strings.EqualFold(role, member.Role) {
				isAuth = true
			}
		}
		if groupID != primitive.NilObjectID && isAuth {
			// If member belongs to the group, continue to the next handler
			return c.Next()
		}

		// If member does not belong to the group, return an error response
		return response.SendError(c, fiber.StatusForbidden, "Member not authorized")
	}
}

func CheckChatIsGroup(gUc domain.GroupUC) fiber.Handler {
	// This function returns the middleware handler
	return func(c *fiber.Ctx) error {
		groupID, err := primitive.ObjectIDFromHex(c.Params("groupID"))
		if err != nil {
			return response.SendError(c, fiber.StatusBadRequest, err.Error())
		}
		// Check if the member belongs to the group
		group, err := gUc.GetGroupById(c.Context(), groupID)
		if err != nil {
			return response.SendError(c, fiber.StatusForbidden, "Group not found")
		}
		if groupID != primitive.NilObjectID && group.IsGroup {
			// If member belongs to the group, continue to the next handler
			return c.Next()
		}

		// If member does not belong to the group, return an error response
		return response.SendError(c, fiber.StatusMethodNotAllowed, "Operation not allowed")
	}
}
