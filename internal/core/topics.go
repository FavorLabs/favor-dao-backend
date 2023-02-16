package core

import (
	"favor-dao-backend/internal/model"
)

type TopicService interface {
	CreateTag(tag *model.Tag) (*model.Tag, error)
	DeleteTag(tag *model.Tag) error
	GetTags(conditions *model.ConditionsT, offset, limit int) ([]*model.Tag, error)
	GetTagsByKeyword(keyword string) ([]*model.Tag, error)
}
