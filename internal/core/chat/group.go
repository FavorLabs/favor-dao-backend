package chat

import "favor-dao-backend/internal/model/chat"

type ManageService interface {
	LinkDao(m *chat.Group) (*chat.Group, error)
	UnlinkDao(m *chat.Group) error
	FindGroupByDao(daoId string) (*chat.Group, error)
}
