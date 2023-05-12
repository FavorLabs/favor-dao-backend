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

func UseTransaction(ctx context.Context, db *mongo.Database, fn func(ctx context.Context) error) error {
	return db.Client().UseSession(ctx, func(sCtx mongo.SessionContext) error {
		err := sCtx.StartTransaction()
		if err != nil {
			return fmt.Errorf("start transaction %v \n", err)
		}
		err = fn(sCtx)
		if err != nil {
			errs := sCtx.AbortTransaction(sCtx)
			if errs != nil {
				return fmt.Errorf("abort transaction %v \n", errs)
			}
			return fmt.Errorf("execute transaction %v \n", err)
		}
		err = sCtx.CommitTransaction(sCtx)
		if err != nil {
			return fmt.Errorf("commit transactions %v \n", err)
		}
		return nil
	})
}
