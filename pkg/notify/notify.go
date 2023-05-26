package notify

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

func (g *Gateway) request(ctx context.Context, method, url string, body, respData interface{}) error {
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
	resp, err := g.client.Do(req)
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

func (g *Gateway) Notify(ctx context.Context, notify PushNotifyRequest) error {
	u := g.baseUrl + "/v1/push/notify"
	var resp BaseResponse
	err := g.request(ctx, http.MethodPost, u, notify, &resp)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}
	return nil
}

func (g *Gateway) NotifySys(ctx context.Context, notify PushNotifySysRequest) error {
	u := g.baseUrl + "/v1/push/notify/sys"
	var resp BaseResponse
	err := g.request(ctx, http.MethodPost, u, notify, &resp)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return errors.New(resp.Msg)
	}
	return nil
}
