package pointSystem

type BaseResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
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
