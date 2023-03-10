package core

import (
	"favor-dao-backend/internal/model"
)

type DaoManageService interface {
	GetDaoByKeyword(keyword string) ([]*model.Dao, error)
	GetDao(dao *model.Dao) (*model.Dao, error)
	GetDaoByName(dao *model.Dao) (*model.DaoFormatted, error)
	GetMyDaoList(dao *model.Dao) ([]*model.DaoFormatted, error)
	CreateDao(dao *model.Dao) (*model.Dao, error)
	UpdateDao(dao *model.Dao) error
	DaoBookmarkCount(address string) int64
	GetDaoBookmarkList(userAddress string, q *QueryReq, offset, limit int) (list []*model.DaoFormatted)
	GetDaoBookmarkListByAddress(address string) []*model.DaoBookmark
	GetDaoBookmarkByAddressAndDaoID(myAddress string, daoId string) (*model.DaoBookmark, error)
	CreateDaoFollow(myAddress string, daoID string) (*model.DaoBookmark, error)
	DeleteDaoFollow(d *model.DaoBookmark) error
	GetDaoCount(conditions *model.ConditionsT) (int64, error)
	GetDAOs(conditions *model.ConditionsT, offset, limit int) ([]*model.Dao, error)
}
