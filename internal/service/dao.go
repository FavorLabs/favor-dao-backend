package service

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DaoCreationReq struct {
	Name         string            `json:"name"          binding:"required"`
	Introduction string            `json:"introduction"`
	Visibility   model.DaoVisibleT `json:"visibility"`
}

func CreateDao(c *gin.Context, userAddress string, param DaoCreationReq) (_ *model.DaoFormatted, err error) {
	dao := &model.Dao{
		Address:      userAddress,
		Name:         param.Name,
		Visibility:   param.Visibility,
		Introduction: param.Introduction,
	}
	res, err := dao.Create(c.Request.Context(), db)
	if err != nil {
		return nil, err
	}

	if dao.Visibility != model.DaoVisitPrivate {
		// create first post
		_, err = CreatePost(c, userAddress, PostCreationReq{
			Contents: []*PostContentItem{{
				Content: "I created a new DAO, welcome to follow!",
				Type:    model.CONTENT_TYPE_TEXT,
			}},
			Tags: []string{"新人报到"},
		})
		if err != nil {
			logrus.Warnf("%s create first post err: %v", userAddress, err)
		}
	}

	return res.Format(), nil
}

func GetDaoBookmarkList(c *gin.Context, userAddress string, q *core.QueryReq, offset, limit int) (list []*model.DaoFormatted, total int64) {
	query := bson.M{"address": userAddress}
	dao := &model.Dao{}
	pipeline := mongo.Pipeline{
		{{"$match", query}},
		{{"$lookup", bson.M{
			"from":         dao.Table(),
			"localField":   "dao_id",
			"foreignField": "_id",
			"as":           "dao",
		}}},
		{{"$skip", offset}},
		{{"$limit", limit}},
		{{"$unwind", "$dao"}},
		{{"$sort", bson.M{"_id": -1}}},
	}
	book := &model.DaoBookmark{Address: userAddress}
	list = book.GetList(c.Request.Context(), db, pipeline)
	total = book.CountMark(c.Request.Context(), db)
	return list, total
}
