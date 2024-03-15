package controller

import (
	"errors"

	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/gofiber/fiber/v2"
)

func GetUserFromReq(c *fiber.Ctx) (domain.User, error) {
	userInfo, ok := c.Locals("userInfo").(domain.User)
	if !ok {
		return userInfo, errors.New("unable to authenticate user")
	}
	return userInfo, nil
}
