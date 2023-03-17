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
	"favor-dao-backend/internal/model"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var chatClient = &http.Client{}

type requestResponseWriter struct {
	gin.ResponseWriter
	reset    bool
	request  []byte
	response *bytes.Buffer
}

func (r *requestResponseWriter) Flush() {
	r.reset = false
	r.ResponseWriter.Write(r.response.Bytes())
}

func (r *requestResponseWriter) Write(b []byte) (int, error) {
	if r.reset {
		r.response.Reset()
	}

	return r.response.Write(b)
}

var cachedBodyKey = "cached-body"

func cacheResponse(c *gin.Context) {
	if _, exists := c.Get(cachedBodyKey); !exists {
		w := &requestResponseWriter{request: make([]byte, c.Request.ContentLength), response: &bytes.Buffer{}, ResponseWriter: c.Writer}
		buf, _ := io.ReadAll(c.Request.Body)
		copy(w.request, buf)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(buf))
		c.Writer = w
		c.Set(cachedBodyKey, w)
	}
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
	req, _ := http.NewRequest(method, chatApi(url), body)
	req.Header.Set("X-Session-Token", token)
	if setJson {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func chatLogin(bodyWriter *requestResponseWriter) (string, bool) {
	req, _ := http.NewRequest(http.MethodGet, chatApi("onboard/hello"), nil)
	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(bodyWriter.response.Bytes(), &body)
	req.Header.Set("X-Session-Token", body.Data.Token)
	resp, err := chatClient.Do(req)
	if err != nil {
		logrus.Errorf("chatLogin: doRequest: %s", err)
		return body.Data.Token, true
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		var respBody []byte
		_, err := resp.Body.Read(respBody)
		if err != nil {
			logrus.Errorf("chatLogin: recv body: %s", err)
		} else {
			logrus.Errorf("chatLogin: %s", string(respBody))
		}
		bodyWriter.reset = true
		return body.Data.Token, true
	}
	return "", false
}

func recoverChatLogin(c *gin.Context, token string) {
	// clean user session
	_ = conf.Redis.Del(c, fmt.Sprintf("token_%s", token))
	app.NewResponse(c).ToErrorResponse(errcode.UnauthorizedTokenError)
}

type createServer struct {
	Name string `json:"name"`
}

func chatCreateServer(c *gin.Context, bodyWriter *requestResponseWriter) (string, bool) {
	var body struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
	}
	_ = json.Unmarshal(bodyWriter.response.Bytes(), &body)
	jsonStr, _ := json.Marshal(createServer{
		Name: body.Data.Name,
	})
	req, _ := createRequest(c, http.MethodPost, "servers/create", jsonStr)
	resp, err := chatClient.Do(req)
	if err != nil {
		logrus.Errorf("chatCreateServer: doRequest: %s", err)
		return body.Data.ID, true
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		var respBody []byte
		_, err := resp.Body.Read(respBody)
		if err != nil {
			logrus.Fatalf("chatCreateServer: recv body: %s", err)
		} else {
			logrus.Fatalf("chatCreateServer: %s", string(respBody))
		}
		bodyWriter.reset = true
		return body.Data.ID, true
	}
	return "", false
}

func recoverChatCreateServer(c *gin.Context, daoId string) {
	// remove created dao
	id, _ := primitive.ObjectIDFromHex(daoId)
	dao := &model.Dao{
		ID: id,
	}
	if err := dao.Delete(c, conf.MustMongoDB()); err != nil {
		logrus.Errorf("recoverChatCreateServer: delete dao: %s", err)
	}
	app.NewResponse(c).ToErrorResponse(errcode.CreateDaoFailed)
}

func chatJoinOrLeaveServer(c *gin.Context, bodyWriter *requestResponseWriter) (string, bool) {
	params := service.DaoFollowReq{}
	_ = json.Unmarshal(bodyWriter.request, &params)
	var body struct {
		Data struct {
			Status bool `json:"status"`
		}
	}
	_ = json.Unmarshal(bodyWriter.response.Bytes(), &body)
	var (
		req *http.Request
		err error
	)
	dao, err := service.GetDao(params.DaoID)
	if err != nil {
		logrus.Errorf("chatJoinOrLeaveServer: getDao(%s): %s", params.DaoID, err)
		return params.DaoID, true
	}
	if body.Data.Status {
		req, _ = createRequest(c, http.MethodPost, fmt.Sprintf("invites/%s", dao.Name), nil)
	} else {
		hashName := crypto.Keccak256([]byte(fmt.Sprintf("server_%s", dao.Name)))
		req, _ = createRequest(c, http.MethodDelete, fmt.Sprintf("servers/%s", hex.EncodeToString(hashName)), nil)
	}
	resp, err := chatClient.Do(req)
	if err != nil {
		logrus.Errorf("chatJoinOrLeaveServer: doRequest: %s", err)
		return params.DaoID, true
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 300 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.Errorf("chatJoinOrLeaveServer: recv body: %s", err)
		} else {
			logrus.Errorf("chatJoinOrLeaveServer: %s", string(respBody))
		}
		bodyWriter.reset = true
		return params.DaoID, true
	}
	return params.DaoID, false
}

func recoverChatBookmark(c *gin.Context, daoId string) {
	// remove created dao
	id, _ := primitive.ObjectIDFromHex(daoId)
	book := &model.DaoBookmark{ID: id}
	if err := book.Delete(c, conf.MustMongoDB()); err != nil {
		logrus.Errorf("recoverChatCreateServer: delete dao: %s", err)
	}
	app.NewResponse(c).ToResponse(gin.H{
		"status": false,
	})
}

func Aggregate() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		var bodyWriter *requestResponseWriter
		if val, _ := c.Get(cachedBodyKey); val != nil {
			bodyWriter = val.(*requestResponseWriter)
			defer bodyWriter.Flush()
		}

		if c.Writer.Status() == http.StatusOK {
			switch strings.TrimPrefix(c.Request.URL.Path, "/v1") {
			case "/auth/login_hello":
				token, failed := chatLogin(bodyWriter)
				if failed {
					recoverChatLogin(c, token)
				}
			case "/dao_server":
				if c.Request.Method == http.MethodPost {
					id, failed := chatCreateServer(c, bodyWriter)
					if failed {
						recoverChatCreateServer(c, id)
					}
				}
			case "/dao/bookmark_server":
				if c.Request.Method == http.MethodPost {
					id, failed := chatJoinOrLeaveServer(c, bodyWriter)
					if failed {
						recoverChatBookmark(c, id)
					}
				}
			}
		}
	}
}
