package model

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConditionsT map[string]bson.M

func create(ctx context.Context, db *mongo.Database, c Collection, opts ...*options.InsertOneOptions) error {
	if err := callToBeforeCreateHooks(ctx, c); err != nil {
		return err
	}

	res, err := db.Collection(c.Table()).InsertOne(ctx, c, opts...)
	if err != nil {
		return err
	}

	// Save ID after insert
	c.SetID(res.InsertedID)

	return callToAfterCreateHooks(ctx, c)
}

func find(ctx context.Context, db *mongo.Database, c Collection, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	return db.Collection(c.Table()).Find(ctx, filter, opts...)
}

func findOne(ctx context.Context, db *mongo.Database, c Collection, filter interface{}, opts ...*options.FindOneOptions) error {
	return db.Collection(c.Table()).FindOne(ctx, filter, opts...).Decode(c)
}

func update(ctx context.Context, db *mongo.Database, c Collection, opts ...*options.UpdateOptions) error {
	if err := callToBeforeUpdateHooks(ctx, c); err != nil {
		return err
	}

	res, err := db.Collection(c.Table()).UpdateOne(ctx, bson.M{"_id": c.GetID()}, bson.M{"$set": c}, opts...)
	if err != nil {
		return err
	}

	return callToAfterUpdateHooks(ctx, res, c)
}

func findAndUpdate(ctx context.Context, db *mongo.Database, c Collection, filter, update interface{}, opts ...*options.FindOneAndUpdateOptions) error {
	res := db.Collection(c.Table()).FindOneAndUpdate(ctx, filter, update, opts...)
	return res.Err()
}

func remove(ctx context.Context, db *mongo.Database, c interface{}, must bool) error {
	switch tc := c.(type) {
	case Collection:
		if err := callToBeforeDeleteHooks(ctx, tc); err != nil {
			return err
		}

		res, err := db.Collection(tc.Table()).DeleteOne(ctx, bson.M{"_id": tc.GetID()})
		if err != nil {
			return err
		}

		return callToAfterDeleteHooks(ctx, res, tc)
	case CollectionMarkDeleting:
		if must {
			if err := callToBeforeDeleteHooks(ctx, tc); err != nil {
				return err
			}

			res, err := db.Collection(tc.Table()).DeleteOne(ctx, bson.M{"_id": tc.GetID()})
			if err != nil {
				return err
			}

			return callToAfterDeleteHooks(ctx, res, tc)
		} else {
			if err := tc.Mark(ctx); err != nil {
				return err
			}

			res := db.Collection(tc.Table()).FindOneAndUpdate(ctx, bson.M{"_id": tc.GetID()}, tc.(MarkDeleting))
			return res.Err()
		}
	default:
		panic("not collection")
	}
}

func UseTransaction(ctx context.Context, db *mongo.Database, fn func(sessCtx mongo.SessionContext) (interface{}, error)) (interface{}, error) {
	session, err := db.Client().StartSession()
	if err != nil {
		return nil, fmt.Errorf("start session: %v", err)
	}
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, fn)
	if err != nil {
		return nil, err
	}

	return result, err
}
