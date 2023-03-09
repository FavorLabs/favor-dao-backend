package service

import (
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DaoCreationReq struct {
	Name         string            `json:"name"          binding:"required"`
	Introduction string            `json:"introduction"`
	Visibility   model.DaoVisibleT `json:"visibility"`
}

type DaoUpdateReq struct {
	Id           primitive.ObjectID `json:"id"            binding:"required"`
	Name         string             `json:"name"          binding:"required"`
	Introduction string             `json:"introduction"`
	Visibility   model.DaoVisibleT  `json:"visibility"`
	Avatar       string             `json:"avatar"`
}

type DaoFollowReq struct {
	DaoID string `json:"dao_id" binding:"required"`
}

func GetDaoByName(name string) (_ *model.DaoFormatted, err error) {
	dao := &model.Dao{
		Name: name,
	}
	return ds.GetDaoByName(dao)
}

func CreateDao(c *gin.Context, userAddress string, param DaoCreationReq) (_ *model.DaoFormatted, err error) {
	dao := &model.Dao{
		Address:      userAddress,
		Name:         param.Name,
		Visibility:   param.Visibility,
		Introduction: param.Introduction,
	}
	res, err := ds.CreateDao(dao)
	if err != nil {
		return nil, err
	}

	if dao.Visibility == model.DaoVisitPublic {
		// create first post
		_, err = CreatePost(c, userAddress, PostCreationReq{
			Visibility: model.PostVisitPublic,
			DaoId:      dao.ID,
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

func GetDaoBookmarkList(userAddress string, q *core.QueryReq, offset, limit int) (list []*model.DaoFormatted, total int64) {
	list = ds.GetDaoBookmarkList(userAddress, q, offset, limit)
	// get avatar
	var addresses []string
	for _, v := range list {
		addresses = append(addresses, v.Address)
	}
	users, err := ds.GetUsersByAddresses(addresses)
	if err != nil {
		return
	}
	userMap := make(map[string]*model.User, len(users))
	for _, user := range users {
		userMap[user.Address] = user
	}
	for k, v := range list {
		if list[k].Avatar == "" {
			list[k].Avatar = userMap[v.Address].Avatar
		}
	}
	if len(list) > 0 {
		total = ds.DaoBookmarkCount(userAddress)
	}
	return list, total
}

func GetDaoBookmarkListByAddress(address string) *[]primitive.ObjectID {
	list := ds.GetDaoBookmarkListByAddress(address)
	daoIds := make([]primitive.ObjectID, 0, len(list))
	for _, l := range list {
		daoIds = append(daoIds, l.DaoID)
	}
	return &daoIds
}

func UpdateDao(userAddress string, param DaoUpdateReq) (err error) {
	dao := &model.Dao{
		ID:           param.Id,
		Name:         param.Name,
		Visibility:   param.Visibility,
		Introduction: param.Introduction,
	}
	getDao, err := ds.GetDao(dao)
	if err != nil {
		return err
	}
	if getDao.Address != userAddress {
		return errcode.NoPermission
	}
	return ds.UpdateDao(dao)
}

func GetDao(daoId string) (*model.DaoFormatted, error) {
	id, err := primitive.ObjectIDFromHex(daoId)
	if err != nil {
		return nil, err
	}
	dao := &model.Dao{
		ID: id,
	}
	res, err := ds.GetDao(dao)
	if err != nil {
		return nil, err
	}
	return res.Format(), nil
}

func GetMyDaoList(address string) ([]*model.DaoFormatted, error) {
	dao := &model.Dao{
		Address: address,
	}
	return ds.GetMyDaoList(dao)
}

func GetDaoBookmark(userAddress string, daoId string) (*model.DaoBookmark, error) {
	return ds.GetDaoBookmarkByAddressAndDaoID(userAddress, daoId)
}

func CreateDaoBookmark(myAddress string, daoId string) (*model.DaoBookmark, error) {
	return ds.CreateDaoFollow(myAddress, daoId)
}

func DeleteDaoBookmark(book *model.DaoBookmark) error {
	return ds.DeleteDaoFollow(book)
}
