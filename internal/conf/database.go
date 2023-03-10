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
	UniqueIndexes []bson.M
	ShardCount    []int
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
		if len(table.ShardCount) > 0 {
			for j := 0; j < table.ShardCount[0]; j++ {
				for k := 0; k < table.ShardCount[1]; k++ {
					name := fmt.Sprintf("%s_%02d%02d", table.TableName, j, k)
					err = fun(name)
					if err != nil {
						return err
					}
				}
			}
		} else {
			err = fun(table.TableName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func createIndexAll() error {
	for i := 0; i < len(tables); i++ {
		tb := tables[i]
		if len(tb.ShardCount) > 0 {
			prefix := tb.TableName
			for j := 0; j < tb.ShardCount[0]; j++ {
				for k := 0; k < tb.ShardCount[1]; k++ {
					name := fmt.Sprintf("%s_%02d%02d", prefix, j, k)
					tb.TableName = name
					err := createIndex(tb)
					if err != nil {
						return err
					}
				}
			}
			tb.TableName = prefix
		} else {
			err := createIndex(tb)
			if err != nil {
				return err
			}
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
				for key, v := range params[j] {
					tmp = append(tmp, bson.E{Key: key, Value: gconv.Int(v)})
				}
			}
			idx[i] = mongo.IndexModel{Keys: tmp}
		}
		_, err := collection.Indexes().CreateMany(context.TODO(), idx)
		if err != nil {
			return fmt.Errorf("table: %s , create index %s", tableInfo.TableName, err)
		}
	}
	for _, v := range tableInfo.UniqueIndexes {
		tmp := bson.D{}
		for key, value := range v {
			if _, ok := value.(string); !ok {
				tmp = append(tmp, bson.E{Key: key, Value: gconv.Int(value)})
			} else {
				tmp = append(tmp, bson.E{Key: key, Value: value})
			}
		}
		var unique = true
		_, err := collection.Indexes().CreateOne(context.TODO(), mongo.IndexModel{Keys: tmp, Options: &options.IndexOptions{Unique: &unique}})
		if err != nil {
			return fmt.Errorf("table: %s , create unique index key(%v) %s", tableInfo.TableName, v, err)
		}
	}
	return nil
}
