package model

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type Tag struct {
	*Model
	Address  string `json:"address"`
	Tag      string `json:"tag"`
	QuoteNum int64  `json:"quote_num"`
}
type TagFormated struct {
	ID       int64         `json:"id"`
	Address  string        `json:"address"`
	User     *UserFormated `json:"user"`
	Tag      string        `json:"tag"`
	QuoteNum int64         `json:"quote_num"`
}

func (t *Tag) Format() *TagFormated {
	if t.Model == nil {
		return &TagFormated{}
	}

	return &TagFormated{
		ID:       t.ID,
		Address:  t.Address,
		User:     &UserFormated{},
		Tag:      t.Tag,
		QuoteNum: t.QuoteNum,
	}
}

func (t *Tag) Get(db *mongo.Database) (*Tag, error) {
	var tag Tag
	if t.Model != nil && t.Model.ID > 0 {
		db = db.Where("id= ? AND is_del = ?", t.Model.ID, 0)
	} else {
		db = db.Where("tag = ? AND is_del = ?", t.Tag, 0)
	}

	err := db.First(&tag).Error
	if err != nil {
		return &tag, err
	}

	return &tag, nil
}

func (t *Tag) Create(db *mongo.Database) (*Tag, error) {
	err := db.Create(&t).Error

	return t, err
}

func (t *Tag) Update(db *mongo.Database) error {
	return db.Model(&Tag{}).Where("id = ? AND is_del = ?", t.Model.ID, 0).Save(t).Error
}

func (t *Tag) Delete(db *mongo.Database) error {
	return db.Model(t).Where("id = ?", t.Model.ID).Updates(map[string]interface{}{
		"deleted_on": time.Now().Unix(),
		"is_del":     1,
	}).Error
}

func (t *Tag) List(db *mongo.Database, conditions *ConditionsT, offset, limit int) ([]*Tag, error) {
	var tags []*Tag
	var err error
	if offset >= 0 && limit > 0 {
		db = db.Offset(offset).Limit(limit)
	}
	if t.UserID > 0 {
		db = db.Where("user_id = ?", t.UserID)
	}
	for k, v := range *conditions {
		if k == "ORDER" {
			db = db.Order(v)
		} else {
			db = db.Where(k, v)
		}
	}

	if err = db.Where("is_del = ?", 0).Find(&tags).Error; err != nil {
		return nil, err
	}

	return tags, nil
}

func (t *Tag) TagsFrom(db *mongo.Database, tags []string) (res []*Tag, err error) {
	err = db.Where("tag IN ?", tags).Find(&res).Error
	return
}
