package pointSystem

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"favor-dao-backend/pkg/json"
)

type Gateway struct {
	baseUrl string
	client  *http.Client
}

func New(baseUrl string) *Gateway {
	return &Gateway{
		baseUrl: strings.TrimRight(baseUrl, "/"),
		client:  http.DefaultClient,
	}
}

func (s *Gateway) request(ctx context.Context, method, url string, body, respData interface{}) error {
	var reqBody io.Reader
	if body != nil {
		rawBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(rawBody)
	} else {
		reqBody = nil
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if resp != nil {
			_ = resp.Body.Close()
		}
	}()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var restErr BaseResponse

		err = json.Unmarshal(rawBody, &restErr)
		if err != nil {
			return fmt.Errorf("%s,%s", resp.Status, err)
		}
		return fmt.Errorf("code:%d msg:%s details:%s", restErr.Code, restErr.Msg, restErr.Details)
	}
	return json.Unmarshal(rawBody, respData)
}

func (s *Gateway) Pay(ctx context.Context, param PayRequest) (txID string, err error) {
	url := s.baseUrl + "/v1/pay"

	var resp struct {
		BaseResponse
		Data PayResponse `json:"data,omitempty"`
	}
	err = s.request(ctx, http.MethodPost, url, param, &resp)
	if err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", errors.New(resp.Msg)
	}
	return resp.Data.Id, nil
}

func (s *Gateway) FindAccounts(ctx context.Context, uid string) (list []Account, err error) {
	url := s.baseUrl + "/v1/accounts?uid=" + uid

	var resp struct {
		BaseResponse
		Data struct {
			List  []Account `json:"list"`
			Pager Pager     `json:"pager"`
		} `json:"data,omitempty"`
	}
	err = s.request(ctx, http.MethodGet, url, nil, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Msg)
	}
	return resp.Data.List, nil
}

func (s *Gateway) Transaction() (txID string, err error) {
	return
}

func (s *Gateway) TradeInfo(txId string) {

}
