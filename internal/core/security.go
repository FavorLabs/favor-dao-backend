package core

import (
	"favor-dao-backend/internal/model"
)

type SecurityService interface {
	GetLatestPhoneCaptcha(phone string) (*model.Captcha, error)
	UsePhoneCaptcha(captcha *model.Captcha) error
	SendPhoneCaptcha(phone string) error
}

type AttachmentCheckService interface {
	CheckAttachment(uri string) error
}
