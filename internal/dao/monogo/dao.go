package monogo

import (
	"context"
	"strings"
	"time"

	"favor-dao-backend/internal/core"
	"favor-dao-backend/internal/model"
	chatModel "favor-dao-backend/internal/model/chat"
	"favor-dao-backend/pkg/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	_ core.DaoManageService = (*daoManageServant)(nil)
)

type daoManageServant struct {
	db *mongo.Database
}

func newDaoManageService(db *mongo.Database) core.DaoManageService {
	return &daoManageServant{
		db: db,
	}
}

func (s *daoManageServant) GetDaoByKeyword(keyword string) ([]*model.Dao, error) {
	dao := &model.Dao{}
	keyword = strings.Trim(keyword, " ")
	return dao.FindListByKeyword(context.TODO(), s.db, keyword, 0, 6)
}

func (s *daoManageServant) CreateDao(dao *model.Dao, chatAction func(*model.Dao) (string, error)) (*model.Dao, error) {
	err := util.MongoTransaction(context.TODO(), s.db, func(ctx context.Context) error {
		newDao, err := dao.Create(ctx, s.db)
		if err != nil {
			return err
		}
		groupId, err := chatAction(dao)
		if err != nil {
			return err
		}
		group := &chatModel.Group{
			ID: chatModel.GroupID{
				DaoID:   newDao.ID,
				GroupID: groupId,
				OwnerID: newDao.Address,
			},
		}
		_, err = group.Create(ctx, s.db)
		if err != nil {
			return err
		}
		dao.ID = newDao.ID
		return nil
	})

	if err != nil {
		return nil, err
	}

	return dao, nil
}

func (s *daoManageServant) UpdateDao(dao *model.Dao) error {
	return dao.Update(context.TODO(), s.db)
}

func (s *daoManageServant) DeleteDao(dao *model.Dao) error {
	return dao.Delete(context.TODO(), s.db)
}

func (s *daoManageServant) GetDaoCount(conditions *model.ConditionsT) (int64, error) {
	return (&model.Dao{}).Count(s.db, conditions)
}

func (s *daoManageServant) GetDAOs(conditions *model.ConditionsT, offset, limit int) ([]*model.Dao, error) {
	return (&model.Dao{}).List(s.db, conditions, offset, limit)
}

func (s *daoManageServant) GetDaoByName(dao *model.Dao) (*model.DaoFormatted, error) {
	return dao.GetByName(context.TODO(), s.db)
}

func (s *daoManageServant) GetDao(dao *model.Dao) (*model.Dao, error) {
	return dao.Get(context.TODO(), s.db)
}

func (s *daoManageServant) GetMyDaoList(dao *model.Dao) ([]*model.DaoFormatted, error) {
	return dao.GetListByAddress(context.TODO(), s.db)
}

func (s *daoManageServant) DaoBookmarkCount(address string) int64 {
	book := &model.DaoBookmark{Address: address}
	return book.CountMark(context.TODO(), s.db)
}

func (s *daoManageServant) GetDaoBookmarkList(userAddress string, q *core.QueryReq, offset, limit int) (list []*model.DaoFormatted) {
	query := bson.M{
		"address": userAddress,
		"is_del":  0,
	}
	dao := &model.Dao{}
	if q.Query != "" {
		query[dao.Table()+".name"] = bson.M{"$regex": q.Query}
	}
	if len(q.Addresses) > 0 {
		query[dao.Table()+".address"] = bson.M{"$in": q.Addresses}
	}
	pipeline := mongo.Pipeline{
		{{"$lookup", bson.M{
			"from":         dao.Table(),
			"localField":   "dao_id",
			"foreignField": "_id",
			"as":           "dao",
		}}},
		{{"$match", query}},
		{{"$sort", bson.M{"_id": -1}}},
		{{"$skip", offset}},
		{{"$limit", limit}},
		{{"$unwind", "$dao"}},
	}
	book := &model.DaoBookmark{Address: userAddress}
	list = book.GetList(context.TODO(), s.db, pipeline)
	return
}

func (s *daoManageServant) GetDaoBookmarkListByAddress(address string) []*model.DaoBookmark {
	book := &model.DaoBookmark{}
	return book.FindList(context.TODO(), s.db, bson.M{"address": address})
}

func (s *daoManageServant) GetDaoBookmarkByAddressAndDaoID(myAddress string, daoId string) (*model.DaoBookmark, error) {
	book := &model.DaoBookmark{}
	res, err := book.GetByAddress(context.TODO(), s.db, myAddress, daoId)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *daoManageServant) CreateDaoFollow(myAddress string, daoID string, chatAction func(context.Context, string) (string, error)) (out *model.DaoBookmark, err error) {
	id, err := primitive.ObjectIDFromHex(daoID)
	if err != nil {
		return nil, err
	}

	err = util.MongoTransaction(context.TODO(), s.db, func(ctx context.Context) error {
		dao, err := (&model.Dao{ID: id, IsDel: 1}).Get(ctx, s.db)
		if err != nil {
			return err
		}
		dao.FollowCount++
		err = dao.Update(ctx, s.db)
		if err != nil {
			return err
		}
		// update book
		book := &model.DaoBookmark{Address: myAddress, DaoID: id}
		out, err = book.GetByAddress(ctx, s.db, myAddress, daoID, true)
		if err != nil {
			out, err = book.Create(ctx, s.db)
		} else {
			out.IsDel = 0
			out.ModifiedOn = time.Now().Unix()
			out.DeletedOn = 0
			err = out.Update(ctx, s.db)
		}
		groupId, err := chatAction(ctx, dao.ID.Hex())
		if err != nil {
			return err
		}
		group := &chatModel.Group{
			ID: chatModel.GroupID{
				DaoID:   dao.ID,
				GroupID: groupId,
				OwnerID: myAddress,
			},
		}
		_, err = group.Create(ctx, s.db)
		return err
	})
	return
}

func (s *daoManageServant) DeleteDaoFollow(d *model.DaoBookmark, chatAction func(context.Context, string) (string, error)) error {
	return util.MongoTransaction(context.TODO(), s.db, func(ctx context.Context) error {
		dao, err := (&model.Dao{ID: d.DaoID, IsDel: 1}).Get(ctx, s.db)
		if err != nil {
			return err
		}
		dao.FollowCount--
		err = dao.Update(ctx, s.db)
		if err != nil {
			return err
		}
		err = d.Delete(ctx, s.db)
		if err != nil {
			return err
		}
		groupId, err := chatAction(ctx, dao.ID.Hex())
		if err != nil {
			return err
		}
		group := &chatModel.Group{
			ID: chatModel.GroupID{
				DaoID:   d.DaoID,
				GroupID: groupId,
				OwnerID: d.Address,
			},
		}
		return group.Delete(ctx, s.db)
	})
}
