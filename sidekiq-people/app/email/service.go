package email

import "github.com/TestingSDK2/sidekiq-backend/sidekiq-models/model"

type Service interface {
	SendEmail(email model.Email) error
}

type service struct {
}

func NewService() Service {
	return &service{}
}

func (s *service) SendEmail(email model.Email) error {
	return sendEmail(email)
}
