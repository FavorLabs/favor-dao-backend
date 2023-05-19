package service

import (
	"context"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/convert"
	"favor-dao-backend/pkg/errcode"
	"favor-dao-backend/pkg/pointSystem"
	"favor-dao-backend/pkg/psub"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DaoCreationReq struct {
	Name         string            `json:"name"          binding:"required"`
	Tags         []string          `json:"tags"`
	Introduction string            `json:"introduction"`
	Visibility   model.DaoVisibleT `json:"visibility"`
	Avatar       string            `json:"avatar"`
	Banner       string            `json:"banner"`
	Price        string            `json:"price"`
}

type DaoUpdateReq struct {
	Id           primitive.ObjectID `json:"id"            binding:"required"`
	Tags         []string           `json:"tags"`
	Name         string             `json:"name"`
	Introduction string             `json:"introduction"`
	Visibility   model.DaoVisibleT  `json:"visibility"`
	Avatar       string             `json:"avatar"`
	Banner       string             `json:"banner"`
	Price        string             `json:"price"`
}

type DaoFollowReq struct {
	DaoID string `json:"dao_id" binding:"required"`
}

type DaoListReq struct {
	Conditions model.ConditionsT
	Offset     int
	Limit      int
}

func GetDaoByName(name string) (_ *model.DaoFormatted, err error) {
	dao := &model.Dao{
		Name: name,
	}
	return ds.GetDaoByName(dao)
}

func CreateDao(_ *gin.Context, userAddress string, param DaoCreationReq, chatAction func(context.Context, *model.Dao) (string, error)) (_ *model.DaoFormatted, err error) {
	tags := tagsFrom(param.Tags)
	dao := &model.Dao{
		Tags:         strings.Join(tags, ","),
		Address:      userAddress,
		Name:         param.Name,
		Visibility:   param.Visibility,
		Introduction: param.Introduction,
		Avatar:       param.Avatar,
		Banner:       param.Banner,
		Price:        param.Price,
		FollowCount:  1, // default owner joined
	}
	if param.Price == "" {
		dao.Price = "10000" // default subscribe price
	} else {
		_, err = convert.StrTo(param.Price).BigInt()
		if err != nil {
			return nil, err
		}
	}
	res, err := ds.CreateDao(dao, chatAction)
	if err != nil {
		return nil, err
	}

	for _, t := range tags {
		tag := &model.Tag{
			Address: userAddress,
			Tag:     t,
		}
		ds.CreateTag(tag)
	}
	// push to search
	_, err = PushDaoToSearch(dao)
	if err != nil {
		logrus.Warnf("%s when create, push dao to search err: %v", userAddress, err)
	}

	return res.Format(), nil
}

func DeleteDao(_ *gin.Context, daoId string) error {
	id, _ := primitive.ObjectIDFromHex(daoId)
	return ds.DeleteDao(&model.Dao{ID: id})
}

func GetDaoBookmarkList(userAddress string, q *core.QueryReq, offset, limit int) (list []*model.DaoFormatted, total int64) {
	list = ds.GetDaoBookmarkList(userAddress, q, offset, limit)
	if len(list) > 0 {
		total = ds.DaoBookmarkCount(userAddress)
	}
	return list, total
}

func GetDaoBookmarkListByAddress(address string) []primitive.ObjectID {
	list := ds.GetDaoBookmarkListByAddress(address)
	daoIds := make([]primitive.ObjectID, 0, len(list))
	for _, l := range list {
		daoIds = append(daoIds, l.DaoID)
	}
	return daoIds
}

func GetDaoBookmarkByAddress(address string) []*model.DaoBookmark {
	return ds.GetDaoBookmarkListByAddress(address)
}

func GetDaoBookmarkIDsByAddress(address string) []string {
	list := ds.GetDaoBookmarkListByAddress(address)
	daoIds := make([]string, 0, len(list))
	for _, l := range list {
		daoIds = append(daoIds, l.DaoID.Hex())
	}
	return daoIds
}

func UpdateDao(userAddress string, param DaoUpdateReq) (err error) {
	dao, err := ds.GetDao(&model.Dao{ID: param.Id})
	if err != nil {
		return err
	}
	if dao.Address != userAddress {
		return errcode.NoPermission
	}
	tags := tagsFrom(param.Tags)
	change := false
	if len(tags) != 0 {
		dao.Tags = strings.Join(tags, ",")
		change = true
	}
	if param.Name != "" {
		dao.Name = param.Name
		change = true
	}
	if param.Avatar != "" {
		dao.Avatar = param.Avatar
		change = true
	}
	if param.Introduction != "" {
		dao.Introduction = param.Introduction
		change = true
	}
	if param.Banner != "" {
		dao.Banner = param.Banner
		change = true
	}
	if param.Price != "" {
		_, err = convert.StrTo(param.Price).BigInt()
		if err != nil {
			return err
		}
		dao.Price = param.Price
		change = true
	}
	if dao.Visibility != param.Visibility {
		dao.Visibility = param.Visibility
		change = true
	}
	if !change {
		return errcode.DAONothingChange
	}
	err = ds.UpdateDao(dao, func(ctx context.Context, dao *model.Dao) error {
		return UpdateChatGroup(ctx, dao.Address, dao.ID.Hex(), dao.Name, dao.Avatar, dao.Introduction)
	})
	if err != nil {
		return err
	}
	for _, t := range tags {
		tag := &model.Tag{
			Address: userAddress,
			Tag:     t,
		}
		ds.CreateTag(tag)
	}
	// push to search
	_, err = PushDaoToSearch(dao)
	if err != nil {
		logrus.Warnf("%s when update, push dao to search err: %v", userAddress, err)
	}
	return err
}

func GetDao(daoId string) (*core.Dao, error) {
	id, err := primitive.ObjectIDFromHex(daoId)
	if err != nil {
		return nil, err
	}
	dao := &model.Dao{
		ID: id,
	}
	return ds.GetDao(dao)
}

func GetDaoFormatted(user, daoId string) (*model.DaoFormatted, error) {
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
	out := res.Format()

	out.IsJoined = CheckJoinedDAO(user, id)
	out.IsSubscribed = CheckSubscribeDAO(user, id)

	out.LastPosts = []*model.PostFormatted{}
	// sms
	conditions := model.ConditionsT{
		"query": bson.M{
			"dao_id":     dao.ID,
			"visibility": model.PostVisitPublic,
			"type":       model.SMS,
		},
		"ORDER": bson.M{"_id": -1},
	}
	resp, _ := GetPostList(user, &PostListReq{
		Conditions: &conditions,
		Offset:     0,
		Limit:      1,
	})
	out.LastPosts = append(out.LastPosts, resp...)
	// video
	conditions2 := model.ConditionsT{
		"query": bson.M{
			"dao_id":     dao.ID,
			"visibility": model.PostVisitPublic,
			"type":       model.VIDEO,
		},
		"ORDER": bson.M{"_id": -1},
	}
	resp2, _ := GetPostList(user, &PostListReq{
		Conditions: &conditions2,
		Offset:     0,
		Limit:      1,
	})
	out.LastPosts = append(out.LastPosts, resp2...)
	return out, nil
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

func CreateDaoBookmark(myAddress string, daoId string, chatAction func(context.Context, *model.Dao) (gid string, e error)) (*model.DaoBookmark, error) {
	return ds.CreateDaoFollow(myAddress, daoId, chatAction)
}

func DeleteDaoBookmark(book *model.DaoBookmark, chatAction func(context.Context, *model.Dao) (string, error)) error {
	return ds.DeleteDaoFollow(book, chatAction)
}

func PushDaoToSearch(dao *model.Dao) (bool, error) {
	contentFormatted := dao.Name + "\n"
	contentFormatted += dao.Introduction + "\n"

	tagMaps := map[string]int8{}
	for _, tag := range strings.Split(dao.Tags, ",") {
		tagMaps[tag] = 1
	}

	data := core.DocItems{{
		"id":                dao.ID,
		"address":           dao.Address,
		"dao_id":            dao.ID.Hex(),
		"dao_follow_count":  dao.FollowCount,
		"view_count":        0,
		"collection_count":  0,
		"upvote_count":      0,
		"comment_count":     0,
		"member":            0,
		"visibility":        model.PostVisitPublic, // Only the public dao will enter the search engine
		"is_top":            0,
		"is_essence":        0,
		"content":           contentFormatted,
		"tags":              tagMaps,
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
	totalRows, _ := GetDaoCount(nil)

	pages := math.Ceil(float64(totalRows) / float64(splitNum))
	nums := int(pages)

	for i := 0; i < nums; i++ {
		posts, _ := GetDaoList(&DaoListReq{
			Conditions: model.ConditionsT{},
			Offset:     i * splitNum,
			Limit:      splitNum,
		})

		for _, dao := range posts {
			_, err := PushDaoToSearch(dao)
			if err != nil {
				log.Printf("dao: add document err: %s\n", err)
				continue
			}
			log.Printf("dao: add document success, dao_id: %s\n", dao.ID.Hex())
		}
	}
}

func GetDaoCount(conditions model.ConditionsT) (int64, error) {
	return ds.GetDaoCount(conditions)
}

func GetDaoList(req *DaoListReq) ([]*model.Dao, error) {
	posts, err := ds.GetDaoList(req.Conditions, req.Offset, req.Limit)
	return posts, err
}

func CheckIsMyDAO(address string, daoID primitive.ObjectID) *errcode.Error {
	dao, err := ds.GetDao(&model.Dao{ID: daoID})
	if err != nil {
		return errcode.NoExistDao
	}
	if dao.Address != address {
		return errcode.NoPermission
	}
	return nil
}

func CheckSubscribeDAO(address string, daoID primitive.ObjectID) bool {
	return ds.IsSubscribeDAO(address, daoID)
}

func CheckJoinedDAO(address string, daoID primitive.ObjectID) bool {
	return ds.IsJoinedDAO(address, daoID)
}

func SubDao(ctx context.Context, daoID primitive.ObjectID, address string) (txID string, status core.DaoSubscribeT, err error) {
	var (
		oid    string
		notify *psub.Notify
	)
	defer func() {
		if notify != nil {
			notify.Cancel()
		}
	}()

	// check old subscribe
	sub := model.DaoSubscribe{}
	err = sub.FindOne(ctx, conf.MustMongoDB(), bson.M{"address": address, "dao_id": daoID})
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return
	}
	if err != nil {
		// ErrNoDocuments
		// create order
		err = ds.SubscribeDAO(address, daoID, func(ctx context.Context, orderID string, dao *model.Dao) error {
			oid = orderID
			// sub order
			notify, err = pubsub.NewSubscribe(orderID)
			if err != nil {
				return err
			}
			// pay
			txID, err = point.Pay(ctx, pointSystem.PayRequest{
				FromObject: address,
				ToSubject:  dao.Address,
				Amount:     dao.Price,
				Comment:    "",
				Channel:    "sub_dao",
				ReturnURI:  conf.PointSetting.Callback + "/pay/notify?method=sub_dao&order_id=" + orderID,
				BindOrder:  orderID,
			})
			return err
		})
		if err != nil {
			return
		}
		e := ds.UpdateSubscribeDAOTxID(oid, txID)
		if e != nil {
			logrus.Errorf("ds.UpdateSubscribeDAOTxID order_id:%s tx_id:%s err:%s", oid, txID, e)
			// When an error occurs, wait for the callback to fix the txID again
		}
	} else {
		txID = sub.TxID
		status = sub.Status
		if status != model.DaoSubscribeSubmit {
			return
		}
		// sub order
		oid = sub.ID.Hex()
		notify, _ = pubsub.NewSubscribe(oid)
	}
	// wait pay notify
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case val := <-notify.Ch:
		status = val.(core.DaoSubscribeT)
	}
	return
}

func UpdateSubscribeDAO(orderID, txID string, status model.DaoSubscribeT) error {
	return ds.UpdateSubscribeDAO(orderID, txID, status)
}
