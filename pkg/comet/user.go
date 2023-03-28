package comet

import (
	"encoding/json"
	"net/http"
)

type UserScoped struct {
	*chatRequest
}

type User struct {
	UID       string       `json:"uid"`
	Name      string       `json:"name"`
	Avatar    string       `json:"avatar"`
	Link      string       `json:"link"`
	Role      string       `json:"role"`
	Metadata  UserMetadata `json:"metadata"`
	Tags      []string     `json:"tags"`
	Status    string       `json:"status"`
	CreatedAt int          `json:"createdAt"`
	AuthToken string       `json:"authToken,omitempty"`
}

type userInfo struct {
	UID         string       `json:"uid"`
	Name        string       `json:"name"`
	Avatar      string       `json:"avatar,omitempty"`
	Link        string       `json:"link,omitempty"`
	Role        string       `json:"role,omitempty"`
	Metadata    UserMetadata `json:"metadata"`
	Tags        []string     `json:"tags,omitempty"`
	ReturnToken bool         `json:"withAuthToken,omitempty"`
}

type UserMetadata struct {
	Private struct {
		Email          string `json:"email"`
		ContractNumber string `json:"contractNumber"`
	} `json:"@private"`
}

type UserCreateOption struct {
	Avatar      string
	Link        string
	Role        string
	Metadata    *UserMetadata
	Tags        []string
	ReturnToken bool
}

func (u *UserScoped) Create(uid, name string, opt *UserCreateOption) (*User, error) {
	info := userInfo{
		UID:  uid,
		Name: name,
	}

	if opt != nil {
		info.Avatar = opt.Avatar
		info.Link = opt.Link
		info.Role = opt.Role
		if opt.Metadata != nil {
			info.Metadata = *opt.Metadata
		}
		if len(opt.Tags) > 0 {
			info.Tags = opt.Tags
		}
		info.ReturnToken = opt.ReturnToken
	}

	body, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	req, err := buildRequest(u.setMethod(http.MethodPost).setBody(body))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data struct {
			User
		} `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return &response.Data.User, nil
}

func (u *UserScoped) AuthToken(uid string) *AuthTokenScoped {
	u.setScope("users", uid)
	u.setScope("auth_tokens", "")

	return &AuthTokenScoped{chatRequest: u.chatRequest}
}
