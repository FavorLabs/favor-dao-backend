package pointSystem

type BaseResponse struct {
	Code    int      `json:"code"`
	Msg     string   `json:"msg"`
	Details []string `json:"details,omitempty"`
}

type Pager struct {
	Page      int64 `json:"page"`
	PageSize  int64 `json:"page_size"`
	TotalRows int   `json:"total_rows"`
}

type PayRequest struct {
	FromObject string `json:"from_object" binding:"required"`
	ToSubject  string `json:"to_subject" binding:"required"`
	Amount     string `json:"amount" binding:"required"`
	UseToken   string `json:"use_token,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Channel    string `json:"channel,omitempty"`
	ReturnURI  string `json:"return_uri,omitempty"`
	BindOrder  string `json:"bind_order,omitempty"`
}

type PayResponse struct {
	Id string `json:"id"`
}

type Account struct {
	User      string `json:"ref_id"`
	TokenName string `json:"token_name"`
	Balance   string `json:"balance"`
	Frozen    string `json:"frozen"`
}
