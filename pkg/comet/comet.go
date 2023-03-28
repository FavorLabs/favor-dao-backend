package comet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	ApiGateway = "https://%s.api-%s.cometchat.io/%s"
	ApiVersion = "v3"
)

type ChatGateway struct {
	baseUrl string
	apiKey  string
	client  *http.Client
}

func New(appId, region, apiKey string) *ChatGateway {
	url := fmt.Sprintf(ApiGateway, appId, region, ApiVersion)

	return &ChatGateway{
		baseUrl: url,
		apiKey:  apiKey,
	}
}

func (c *ChatGateway) Scoped() *Scoped {
	req := &chatRequest{
		url:    c.baseUrl,
		scopes: make([][]string, 0),
		header: make(http.Header),
	}

	req.setHeader("accept", "application/json")
	req.setHeader("content-type", "application/json")
	req.setHeader("apikey", c.apiKey)

	return &Scoped{chatRequest: req}
}

type Scoped struct {
	*chatRequest
}

func (s *Scoped) Perform(uid string) *Scoped {
	s.setHeader("onBehalfOf", uid)

	return s
}

func (s *Scoped) Users() *UserScoped {
	return &UserScoped{
		chatRequest: s.setScope("users", ""),
	}
}

func (s *Scoped) Groups() *GroupScoped {
	return &GroupScoped{
		chatRequest: s.setScope("groups", ""),
	}
}

type chatRequest struct {
	url     string
	user    string
	scopes  [][]string
	method  string
	rawBody []byte
	header  http.Header
}

func (r *chatRequest) setScope(key, value string) *chatRequest {
	exists := -1

	for i, scope := range r.scopes {
		if scope[0] == key {
			exists = i
		}
	}

	if exists != -1 {
		r.scopes[exists] = []string{key, value}
	} else {
		r.scopes = append(r.scopes, []string{key, value})
	}

	return r
}

func (r *chatRequest) setBody(body []byte) *chatRequest {
	r.rawBody = body
	return r
}

func (r *chatRequest) setMethod(method string) *chatRequest {
	r.method = method
	return r
}

func (r *chatRequest) setHeader(key, value string) *chatRequest {
	r.header.Set(key, value)
	return r
}

type apiBuilder interface {
	getUrl() string
	getBody() []byte
	getMethod() string
	getHeader() http.Header
}

func (r *chatRequest) getUrl() string {
	var url strings.Builder

	url.WriteString(r.url)
	url.WriteByte('/')

	for _, scope := range r.scopes {
		url.WriteString(scope[0])
		url.WriteByte('/')
		if scope[1] != "" {
			url.WriteString(scope[1])
			url.WriteByte('/')
		}
	}

	return strings.TrimRight(url.String(), "/")
}

func (r *chatRequest) getBody() []byte {
	return r.rawBody
}

func (r *chatRequest) getMethod() string {
	return r.method
}

func (r *chatRequest) getHeader() http.Header {
	return r.header
}

func buildRequest(builder apiBuilder) (*http.Request, error) {
	var body io.Reader
	if rawBody := builder.getBody(); rawBody != nil {
		body = bytes.NewReader(rawBody)
	}

	req, err := http.NewRequest(builder.getMethod(), builder.getUrl(), body)
	if err != nil {
		return nil, err
	}

	if header := builder.getHeader(); header != nil {
		req.Header = header
	}

	return req, nil
}

var apiClient = &http.Client{}

type RestApiError struct {
	Inner struct {
		Message string `json:"message"`
		Detail  string `json:"devMessage"`
		Source  string `json:"source"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (e RestApiError) Error() string {
	return fmt.Sprintf("[%s] %s - %s", e.Inner.Code, e.Inner.Source, e.Inner.Message)
}

type ApiError struct {
	Code int
	Body []byte
}

func (e ApiError) Error() string {
	return fmt.Sprintf("---\n%d\n%s\n---", e.Code, string(e.Body))
}

type ApiResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func doRequest(request *http.Request, body interface{}) error {
	resp, err := apiClient.Do(request)
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

	if resp.StatusCode >= 300 {
		var restErr RestApiError

		err = json.Unmarshal(rawBody, &restErr)
		if err != nil {
			return ApiError{Code: resp.StatusCode, Body: rawBody}
		}

		return restErr
	}

	return json.Unmarshal(rawBody, body)
}
