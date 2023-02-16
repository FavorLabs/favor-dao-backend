package jinzhu

import (
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

func createTag(db *mongo.Database, tag *model.Tag) (*model.Tag, error) {
	t, err := tag.Get(db)
	if err != nil {
		tag.QuoteNum = 1
		return tag.Create(db)
	}

	// update
	t.QuoteNum++
	err = t.Update(db)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func deleteTag(db *mongo.Database, tag *model.Tag) error {
	tag, err := tag.Get(db)
	if err != nil {
		return err
	}
	tag.QuoteNum--
	return tag.Update(db)
}

func deleteTags(db *mongo.Database, tags []string) error {
	allTags, err := (&model.Tag{}).TagsFrom(db, tags)
	if err != nil {
		return err
	}
	for _, tag := range allTags {
		tag.QuoteNum--
		if tag.QuoteNum < 0 {
			tag.QuoteNum = 0
		}
		// Handle errors leniently, update tag records as much as possible, and record only the last error
		if e := tag.Update(db); e != nil {
			err = e
		}
	}
	return err
}

// Get a list of users based on IDs
func getUsersByIDs(db *mongo.Database, ids []int64) ([]*model.User, error) {
	user := &model.User{}
	return user.List(db, &model.ConditionsT{
		"id IN ?": ids,
	}, 0, 0)
}
