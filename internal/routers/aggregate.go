package routers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/service"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var chatClient = &http.Client{}

type requestResponseWriter struct {
	gin.ResponseWriter
	request  []byte
	response *bytes.Buffer
}

func (r requestResponseWriter) Write(b []byte) (int, error) {
	r.response.Write(b)
	return r.ResponseWriter.Write(b)
}

var cachedBodyKey = "cached-body"

func cacheResponse(c *gin.Context) {
	w := &requestResponseWriter{request: make([]byte, c.Request.ContentLength), response: &bytes.Buffer{}, ResponseWriter: c.Writer}
	buf, _ := io.ReadAll(c.Request.Body)
	copy(w.request, buf)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(buf))
	c.Writer = w
	c.Set(cachedBodyKey, w)
	c.Next()
}

func chatApi(path string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(conf.ChatSetting.Api, "/"), strings.TrimLeft(path, "/"))
}

func createRequest(c *gin.Context, method, url string, json []byte) (*http.Request, error) {
	token := c.GetHeader("X-Session-Token")
	var (
		body    io.Reader
		setJson bool
	)
	if json != nil {
		body = bytes.NewBuffer(json)
		setJson = true
	}
	req, err := http.NewRequest(method, chatApi(url), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Session-Token", token)
	if setJson {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func chatLogin(c *gin.Context) {
	req, err := http.NewRequest(http.MethodGet, chatApi("onboard/hello"), nil)
	if err != nil {
		logrus.Fatalf("chatLogin: createRequest: %s", err)
	}
	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	var bodyWriter *requestResponseWriter
	if val, _ := c.Get(cachedBodyKey); val != nil {
		bodyWriter = val.(*requestResponseWriter)
	}
	json.Unmarshal(bodyWriter.response.Bytes(), &body)
	req.Header.Set("X-Session-Token", body.Data.Token)
	resp, err := chatClient.Do(req)
	if err != nil {
		logrus.Fatalf("chatLogin: doRequest: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var body []byte
		_, err := resp.Body.Read(body)
		if err != nil {
			logrus.Fatalf("chatLogin: recv body: %s", err)
		} else {
			logrus.Fatalf("chatLogin: %s", string(body))
		}
	}
}

type createServer struct {
	Name string `json:"name"`
}

func chatCreateServer(c *gin.Context) {
	params := service.DaoCreationReq{}
	var bodyWriter *requestResponseWriter
	if val, _ := c.Get(cachedBodyKey); val != nil {
		bodyWriter = val.(*requestResponseWriter)
	}
	json.Unmarshal(bodyWriter.request, &params)
	jsonStr, _ := json.Marshal(createServer{
		Name: params.Name,
	})
	req, err := createRequest(c, http.MethodPost, "servers/create", jsonStr)
	if err != nil {
		logrus.Fatalf("chatCreateServer: createRequest: %s", err)
	}
	resp, err := chatClient.Do(req)
	if err != nil {
		logrus.Fatalf("chatCreateServer: doRequest: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var body []byte
		_, err := resp.Body.Read(body)
		if err != nil {
			logrus.Fatalf("chatCreateServer: recv body: %s", err)
		} else {
			logrus.Fatalf("chatCreateServer: %s", string(body))
		}
	}
}

func chatJoinOrLeaveServer(c *gin.Context) {
	params := service.DaoFollowReq{}
	var bodyWriter *requestResponseWriter
	if val, _ := c.Get(cachedBodyKey); val != nil {
		bodyWriter = val.(*requestResponseWriter)
	}
	json.Unmarshal(bodyWriter.request, &params)
	var body struct {
		Data struct {
			Status bool `json:"status"`
		}
	}
	json.Unmarshal(bodyWriter.response.Bytes(), &body)
	var (
		req *http.Request
		err error
	)
	dao, err := service.GetDao(params.DaoID)
	if err != nil {
		logrus.Fatalf("chatJoinOrLeaveServer: getDao(%d): %s", params.DaoID, err)
	}
	if body.Data.Status {
		req, err = createRequest(c, http.MethodPost, fmt.Sprintf("invites/%s", dao.Name), nil)
	} else {
		hashName := crypto.Keccak256([]byte(fmt.Sprintf("server_%s", dao.Name)))
		req, err = createRequest(c, http.MethodDelete, fmt.Sprintf("servers/%s", hex.EncodeToString(hashName)), nil)
	}
	if err != nil {
		logrus.Fatalf("chatJoinOrLeaveServer: createRequest: %s", err)
	}
	resp, err := chatClient.Do(req)
	if err != nil {
		logrus.Fatalf("chatJoinOrLeaveServer: doRequest: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.Fatalf("chatJoinOrLeaveServer: recv body: %s", err)
		} else {
			logrus.Fatalf("chatJoinOrLeaveServer: %s", string(respBody))
		}
	}
}

func Aggregate() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() == http.StatusOK {
			switch strings.TrimPrefix(c.Request.URL.Path, "/v1") {
			case "/auth/login_hello":
				chatLogin(c)
			case "/dao_server":
				if c.Request.Method == http.MethodPost {
					chatCreateServer(c)
				}
			case "/dao/bookmark_server":
				if c.Request.Method == http.MethodPost {
					chatJoinOrLeaveServer(c)
				}
			}
		}
	}
}
