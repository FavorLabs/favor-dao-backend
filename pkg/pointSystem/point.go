package pointSystem

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"favor-dao-backend/pkg/json"
)

type Gateway struct {
	baseUrl  string
	callback string
	client   *http.Client
}

func New(baseUrl, callback string) *Gateway {
	return &Gateway{
		baseUrl:  strings.TrimRight(baseUrl, "/"),
		callback: callback,
		client:   http.DefaultClient,
	}
}

func (s *Gateway) request(ctx context.Context, method, url string, body, respData interface{}) error {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	reqBody := bytes.NewReader(rawBody)

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(respBody, respData)
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

func (s *Gateway) Transaction() (txID string, err error) {
	return
}

func (s *Gateway) TradeInfo(txId string) {

}
