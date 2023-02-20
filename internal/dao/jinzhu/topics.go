package jinzhu

import (
	"go.mongodb.org/mongo-driver/bson"
	"strings"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.TopicService = (*topicServant)(nil)
)

type topicServant struct {
	db *mongo.Database
}

func newTopicService(db *mongo.Database) core.TopicService {
	return &topicServant{
		db: db,
	}
}

func (s *topicServant) CreateTag(tag *model.Tag) (*model.Tag, error) {
	return createTag(s.db, tag)
}

func (s *topicServant) DeleteTag(tag *model.Tag) error {
	return deleteTag(s.db, tag)
}

func (s *topicServant) GetTags(conditions *model.ConditionsT, offset, limit int) ([]*model.Tag, error) {
	return (&model.Tag{}).List(s.db, conditions, offset, limit)
}

func (s *topicServant) GetTagsByKeyword(keyword string) ([]*model.Tag, error) {
	tag := &model.Tag{}

	keyword = strings.Trim(keyword, " ")
	if keyword == "" {
		return tag.List(s.db, &model.ConditionsT{
			"ORDER": bson.M{"quote_num": -1},
		}, 0, 6)
	} else {
		return tag.List(s.db, &model.ConditionsT{
			"query": bson.M{"tag": bson.D{{"$all", bson.A{keyword}}}},
			"ORDER": bson.M{"quote_num": -1},
		}, 0, 6)
	}
}
