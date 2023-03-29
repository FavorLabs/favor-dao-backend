package comet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
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
	baseUrl := fmt.Sprintf(ApiGateway, appId, region, ApiVersion)

	return &ChatGateway{
		baseUrl: baseUrl,
		apiKey:  apiKey,
	}
}

func (c *ChatGateway) Scoped() *Scoped {
	req := &chatRequest{
		url:    c.baseUrl,
		scopes: make([][]string, 0),
		query:  make(url.Values),
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
	query   url.Values
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
	if value == "" {
		return r
	}
	r.header.Set(key, value)
	return r
}

func (r *chatRequest) setQuery(key, value string) *chatRequest {
	if value == "" {
		return r
	}
	if r.query.Get(key) == "" {
		r.query.Set(key, value)
	} else {
		r.query.Add(key, value)
	}
	return r
}

func (r *chatRequest) setQueries(values interface{}) *chatRequest {
	rv := reflect.ValueOf(values)
	if rv.Kind() != reflect.Struct {
		return r
	}
	num := rv.NumField()
	rt := rv.Type()
	for i := 0; i < num; i++ {
		f := rv.Field(i)
		name := rt.Field(i).Name
		switch f.Kind() {
		case reflect.String:
			r.setQuery(name, f.String())
		case reflect.Array, reflect.Slice:
			if f.Len() > 0 && f.Index(0).Kind() == reflect.String {
				for j := 0; j < f.Len(); j++ {
					r.setQuery(name, f.Index(j).String())
				}
			}
		default:
			toStr := f.MethodByName("String")
			if toStr.IsValid() {
				retVals := toStr.Call([]reflect.Value{})
				if len(retVals) == 1 && retVals[0].Kind() == reflect.String {
					r.setQuery(name, retVals[0].String())
				}
			}
		}
	}
	return r
}

type apiBuilder interface {
	getUrl() string
	getBody() []byte
	getMethod() string
	getHeader() http.Header
	getQuery() url.Values
}

func (r *chatRequest) getUrl() string {
	var urlStr strings.Builder

	urlStr.WriteString(r.url)
	urlStr.WriteByte('/')

	for _, scope := range r.scopes {
		urlStr.WriteString(scope[0])
		urlStr.WriteByte('/')
		if scope[1] != "" {
			urlStr.WriteString(scope[1])
			urlStr.WriteByte('/')
		}
	}

	return strings.TrimRight(urlStr.String(), "/")
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

func (r *chatRequest) getQuery() url.Values {
	return r.query
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

	if header := builder.getHeader(); len(header) != 0 {
		req.Header = header
	}

	if query := builder.getQuery(); len(query) != 0 {
		req.URL.RawQuery = query.Encode()
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

type Meta struct {
	Pagination Pagination `json:"pagination"`
	Cursor     Cursor     `json:"cursor,omitempty"`
}

type Pagination struct {
	Total       int `json:"total"`
	Count       int `json:"count"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}

type Cursor struct {
	UpdatedAt int    `json:"updated_at"`
	Affix     string `json:"affix"`
}
