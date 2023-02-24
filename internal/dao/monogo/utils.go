package monogo

import (
	"context"

	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
)

func createTag(db *mongo.Database, tag *model.Tag) (*model.Tag, error) {
	t, err := tag.Get(context.TODO(), db)
	if err != nil {
		tag.QuoteNum = 1
		return tag.Create(context.TODO(), db)
	}

	// update
	t.QuoteNum++
	err = t.Update(context.TODO(), db)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func deleteTag(db *mongo.Database, tag *model.Tag) error {
	tag, err := tag.Get(context.TODO(), db)
	if err != nil {
		return err
	}
	tag.QuoteNum--
	return tag.Update(context.TODO(), db)
}

func deleteTags(db *mongo.Database, tags []string) error {
	allTags, err := (&model.Tag{}).TagsFrom(context.TODO(), db, tags)
	if err != nil {
		return err
	}
	for _, tag := range allTags {
		tag.QuoteNum--
		if tag.QuoteNum < 0 {
			tag.QuoteNum = 0
		}
		// Handle errors leniently, update tag records as much as possible, and record only the last error
		if e := tag.Update(context.TODO(), db); e != nil {
			err = e
		}
	}
	return err
}
