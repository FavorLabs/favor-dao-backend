package util

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

func MongoTransaction(ctx context.Context, db *mongo.Database, fn func(ctx context.Context) error) error {
	return db.Client().UseSession(ctx, func(sessionContext mongo.SessionContext) error {
		err := sessionContext.StartTransaction()
		if err != nil {
			return fmt.Errorf("start transaction %v \n", err)
		}
		err = fn(sessionContext)
		if err != nil {
			errs := sessionContext.AbortTransaction(sessionContext)
			if errs != nil {
				return fmt.Errorf("abort transaction %v \n", errs)
			}
			return fmt.Errorf("execute transaction %v \n", err)
		}
		err = sessionContext.CommitTransaction(sessionContext)
		if err != nil {
			return fmt.Errorf("commit transactions %v \n", err)
		}
		return nil
	})
}
