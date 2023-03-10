package service

import (
	"log"
	"math"
	"time"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DaoCreationReq struct {
	Name         string            `json:"name"          binding:"required"`
	Introduction string            `json:"introduction"`
	Visibility   model.DaoVisibleT `json:"visibility"`
	Avatar       string            `json:"avatar"`
	Banner       string            `json:"banner"`
}

type DaoUpdateReq struct {
	Id           primitive.ObjectID `json:"id"            binding:"required"`
	Name         string             `json:"name"          binding:"required"`
	Introduction string             `json:"introduction"`
	Visibility   model.DaoVisibleT  `json:"visibility"`
	Avatar       string             `json:"avatar"`
	Banner       string             `json:"banner"`
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

func CreateDao(_ *gin.Context, userAddress string, param DaoCreationReq) (_ *model.DaoFormatted, err error) {
	dao := &model.Dao{
		Address:      userAddress,
		Name:         param.Name,
		Visibility:   param.Visibility,
		Introduction: param.Introduction,
		Avatar:       param.Avatar,
		Banner:       param.Banner,
	}
	res, err := ds.CreateDao(dao)
	if err != nil {
		return nil, err
	}

	if dao.Visibility == model.DaoVisitPublic {
		// push to search
		_, err = PushDaoToSearch(dao)
		if err != nil {
			logrus.Warnf("%s when create, push dao to search err: %v", userAddress, err)
		}
	}

	return res.Format(), nil
}

func GetDaoBookmarkList(userAddress string, q *core.QueryReq, offset, limit int) (list []*model.DaoFormatted, total int64) {
	list = ds.GetDaoBookmarkList(userAddress, q, offset, limit)
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
		Avatar:       param.Avatar,
		Banner:       param.Banner,
	}
	getDao, err := ds.GetDao(dao)
	if err != nil {
		return err
	}
	if getDao.Address != userAddress {
		return errcode.NoPermission
	}
	if dao.Visibility == model.DaoVisitPublic {
		// push to search
		_, err = PushDaoToSearch(dao)
		if err != nil {
			logrus.Warnf("%s when update, push dao to search err: %v", userAddress, err)
		}
	} else {
		err = DeleteSearchDao(dao)
		if err != nil {
			logrus.Warnf("%s when update, delete dao from search err: %v", userAddress, err)
		}
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

func PushDaoToSearch(dao *model.Dao) (bool, error) {
	contentFormatted := dao.Name + "\n"
	contentFormatted += dao.Introduction + "\n"

	data := core.DocItems{{
		"id":               dao.ID,
		"address":          dao.Address,
		"dao_id":           dao.ID.Hex(),
		"view_count":       0,
		"collection_count": 0,
		"upvote_count":     0,
		"comment_count":    0,
		"member":           0,
		"visibility":       model.PostVisitPublic, // Only the public dao will enter the search engine
		"is_top":           0,
		"is_essence":       0,
		"content":          contentFormatted,
		// "tags":              tagMaps,
		"type":              model.DAO,
		"created_on":        dao.CreatedOn,
		"modified_on":       dao.ModifiedOn,
		"latest_replied_on": time.Now().Unix(),
	}}

	return ts.AddDocuments(data, dao.ID.Hex())
}

func DeleteSearchDao(post *model.Dao) error {
	return ts.DeleteDocuments([]string{post.ID.Hex()})
}

func PushDAOsToSearch() {
	splitNum := 1000
	totalRows, _ := GetDaoCount(&model.ConditionsT{
		"query": bson.M{"visibility": model.DaoVisitPublic},
	})

	pages := math.Ceil(float64(totalRows) / float64(splitNum))
	nums := int(pages)

	for i := 0; i < nums; i++ {
		posts, _ := GetDaoList(&PostListReq{
			Conditions: &model.ConditionsT{},
			Offset:     i * splitNum,
			Limit:      splitNum,
		})

		for _, post := range posts {
			_, err := PushDaoToSearch(post)
			if err != nil {
				log.Printf("dao: add document err: %s\n", err)
				continue
			}
			log.Printf("dao: add document success, dao_id: %s\n", post.ID.Hex())
		}
	}
}

func GetDaoCount(conditions *model.ConditionsT) (int64, error) {
	return ds.GetDaoCount(conditions)
}

func GetDaoList(req *PostListReq) ([]*model.Dao, error) {
	posts, err := ds.GetDAOs(req.Conditions, req.Offset, req.Limit)
	return posts, err
}
