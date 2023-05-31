package conf

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/gogf/gf/util/gconv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var tables []*tableInfo

//go:embed table.json
var tableCfg string

func GetTableInfos() ([]string, error) {
	collectionNames, err := db.ListCollectionNames(context.TODO(), bson.M{})
	return collectionNames, err
}

func CreateTableIndex() (err error) {
	var res []*tableInfo
	err = json.Unmarshal([]byte(tableCfg), &res)
	if err != nil {
		return fmt.Errorf("databse table info error %v \n", err)
	}
	tables = res
	if err = createTable(); err != nil {
		return err
	}
	return createIndexAll()
}

type tableInfo struct {
	TableName     string
	Indexes       []param
	UniqueIndexes []param
}

type param []bson.M

func tableIsExists(name string, collections []string) bool {
	for i := 0; i < len(collections); i++ {
		if collections[i] == name {
			return true
		}
	}
	return false
}

func createTable() error {
	collectionNames, err := GetTableInfos()
	if err != nil {
		return fmt.Errorf("get all collection error %v \n", err)
	}
	var fun = func(name string) error {
		if !tableIsExists(name, collectionNames) {
			err = db.CreateCollection(context.TODO(), name)
			if err != nil {
				return err
			}
		}
		return nil
	}
	for i := 0; i < len(tables); i++ {
		table := tables[i]
		err = fun(table.TableName)
		if err != nil {
			return err
		}
	}
	return nil
}

func createIndexAll() error {
	for i := 0; i < len(tables); i++ {
		tb := tables[i]
		err := createIndex(tb)
		if err != nil {
			return err
		}
	}
	return nil
}

func createIndex(tableInfo *tableInfo) error {
	collection := db.Collection(tableInfo.TableName)
	k := len(tableInfo.Indexes)
	if k > 0 {
		idx := make([]mongo.IndexModel, k)
		for i := 0; i < k; i++ {
			params := tableInfo.Indexes[i]
			tmp := bson.D{}
			for j := 0; j < len(params); j++ {
				for key, value := range params[j] {
					if vInt, ok := value.(string); ok {
						tmp = append(tmp, bson.E{Key: key, Value: vInt})
					} else {
						tmp = append(tmp, bson.E{Key: key, Value: gconv.Int(value)})
					}
				}
			}
			idx[i] = mongo.IndexModel{Keys: tmp}
		}
		_, err := collection.Indexes().CreateMany(context.TODO(), idx)
		if err != nil {
			return fmt.Errorf("table: %s , create index %s", tableInfo.TableName, err)
		}
	}
	k = len(tableInfo.UniqueIndexes)
	if k > 0 {
		idx := make([]mongo.IndexModel, k)
		for i := 0; i < k; i++ {
			params := tableInfo.UniqueIndexes[i]
			tmp := bson.D{}
			for j := 0; j < len(params); j++ {
				for key, value := range params[j] {
					if vInt, ok := value.(string); ok {
						tmp = append(tmp, bson.E{Key: key, Value: vInt})
					} else {
						tmp = append(tmp, bson.E{Key: key, Value: gconv.Int(value)})
					}
				}
			}
			var unique = true
			idx[i] = mongo.IndexModel{Keys: tmp, Options: &options.IndexOptions{Unique: &unique}}
		}
		_, err := collection.Indexes().CreateMany(context.TODO(), idx)
		if err != nil {
			return fmt.Errorf("table: %s , create unique index %s", tableInfo.TableName, err)
		}
	}
	return nil
}
