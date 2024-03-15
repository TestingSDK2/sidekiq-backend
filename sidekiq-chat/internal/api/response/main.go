package response

import (
	"github.com/TestingSDK2/sidekiq-backend/sidekiq-chat/internal/domain"
	"github.com/gofiber/fiber/v2"
)

type AuthResponse struct {
	User    *domain.User
	Profile int
	ErrCode int
	ErrMsg  string
	Error   error
}

// CommonResponse is a struct for common API responses
type CommonResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// SendSuccess sends a success response with optional data
func SendSuccess(c *fiber.Ctx, data interface{}, message string) error {
	response := CommonResponse{
		Success: true,
		Data:    data,
		Message: message,
	}
	return c.JSON(response)
}

// SendError sends an error response with a specified message
func SendError(c *fiber.Ctx, statusCode int, message string) error {
	response := CommonResponse{
		Success: false,
		Message: message,
	}
	return c.Status(statusCode).JSON(response)
}
