package monogo

import (
	"context"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
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
