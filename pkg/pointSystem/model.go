package pointSystem

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type Pager struct {
	Page      int64 `json:"page"`
	PageSize  int64 `json:"page_size"`
	TotalRows int   `json:"total_rows"`
}

type PayRequest struct {
	UseWallet string `json:"use_wallet"          binding:"required"`
	ToSubject string `json:"to_subject"          binding:"required"`
	Amount    int64  `json:"amount"              binding:"required"`
	Decimal   int    `json:"decimal,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Channel   string `json:"channel,omitempty"`
	ReturnURI string `json:"return_uri,omitempty"`
	BindOrder string `json:"bind_order,omitempty"`
}

type PayResponse struct {
	Id string `json:"id"`
}

type CreateAccountRequest struct {
	BindUser   string `json:"bind_user" binding:"required"`
	BindWallet string `json:"bind_wallet"`
}

type Account struct {
	User      primitive.ObjectID `json:"user"`
	Wallet    string             `json:"wallet"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Balance   int64              `json:"balance"`
	Frozen    int64              `json:"frozen"`
	Decimal   int                `json:"decimal"`
}
