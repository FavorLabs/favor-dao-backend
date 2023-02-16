package service

import "favor-dao-backend/internal/model"

func CreateAttachment(attachment *model.Attachment) (*model.Attachment, error) {
	return ds.CreateAttachment(attachment)
}
