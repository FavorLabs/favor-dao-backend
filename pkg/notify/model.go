package notify

import "favor-dao-backend/internal/model"

type PushNotifyRequest struct {
	IsSave    bool               `json:"isSave"`
	From      string             `json:"from"`
	FromType  model.FromTypeEnum `json:"fromType"`
	To        string             `json:"to"`
	Title     string             `json:"title"`
	Content   string             `json:"content"`
	Links     string             `json:"links"`
	NetWorkId int                `json:"netWorkId"`
}

type PushNotifySysRequest struct {
	IsSave    bool   `json:"isSave"`
	From      string `json:"from"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Links     string `json:"links"`
	NetWorkId int    `json:"netWorkId"`
}

type BaseResponse struct {
	Code    int      `json:"code"`
	Msg     string   `json:"msg"`
	Details []string `json:"details,omitempty"`
}
