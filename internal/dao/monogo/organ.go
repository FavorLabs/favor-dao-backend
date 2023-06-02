package monogo

import (
	"context"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.OrganMangerService = (*organManageService)(nil)
)

type organManageService struct {
	db *mongo.Database
}

func newOrganMangerService(db *mongo.Database) core.OrganMangerService {
	return &organManageService{
		db: db,
	}
}

func (o organManageService) GetOrganById(id primitive.ObjectID) (*model.Organ, error) {
	organ := &model.Organ{ID: id}
	return organ.Get(context.TODO(), o.db)
}

func (o organManageService) GetOrganByKey(key string) (*model.Organ, error) {
	organ := &model.Organ{Key: key}
	return organ.Get(context.TODO(), o.db)
}

func (o organManageService) GetOrganNotShow() (*[]primitive.ObjectID, error) {
	organ := &model.Organ{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"isShow": false,
		},
	}
	organs, err := organ.List(o.db, conditions)
	if err != nil {
		return nil, err
	}
	os := *organs
	ids := make([]primitive.ObjectID, 0, len(os))
	for _, or := range os {
		ids = append(ids, or.ID)
	}
	return &ids, err
}

func (o organManageService) ListOrgan() (*[]model.Organ, error) {
	organ := &model.Organ{}
	conditions := &model.ConditionsT{
		"query": bson.M{
			"isShow": false,
		},
		"ORDER": bson.M{"_id": -1},
	}
	return organ.List(o.db, conditions)
}
