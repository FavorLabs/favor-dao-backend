package model

import (
	"time"

	"favor-dao-backend/pkg/types"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/plugin/soft_delete"
)

// Model public
type Model struct {
	ID         int64                 `gorm:"primary_key" json:"id"`
	CreatedOn  int64                 `json:"created_on"`
	ModifiedOn int64                 `json:"modified_on"`
	DeletedOn  int64                 `json:"deleted_on"`
	IsDel      soft_delete.DeletedAt `gorm:"softDelete:flag" json:"is_del"`
}

type ConditionsT map[string]interface{}
type Predicates map[string]types.AnySlice

func (m *Model) BeforeCreate(tx *mongo.Database) (err error) {
	nowTime := time.Now().Unix()

	tx.Statement.SetColumn("created_on", nowTime)
	tx.Statement.SetColumn("modified_on", nowTime)
	return
}

func (m *Model) BeforeUpdate(tx *mongo.Database) (err error) {
	if !tx.Statement.Changed("modified_on") {
		tx.Statement.SetColumn("modified_on", time.Now().Unix())
	}

	return
}
