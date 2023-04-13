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
	Scope     string       `json:"scope,omitempty"`
	Metadata  UserMetadata `json:"metadata"`
	Tags      []string     `json:"tags"`
	Status    string       `json:"status"`
	CreatedAt int          `json:"createdAt"`
	UpdatedAt int          `json:"updatedAt,omitempty"`
	JoinedAt  int          `json:"joinedAt,omitempty"`
	AuthToken string       `json:"authToken,omitempty"`
}

type userInfo struct {
	UID         string       `json:"uid,omitempty"`
	Name        string       `json:"name,omitempty"`
	Avatar      string       `json:"avatar,omitempty"`
	Link        string       `json:"link,omitempty"`
	Role        string       `json:"role,omitempty"`
	Metadata    UserMetadata `json:"metadata"`
	Tags        []string     `json:"tags,omitempty"`
	Unset       []string     `json:"unset,omitempty"`
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

type UserUpdateOption struct {
	Name     string
	Avatar   string
	Link     string
	Role     string
	Metadata *UserMetadata
	Tags     []string
	Unset    []string
}

func (u *UserScoped) Update(uid string, opt UserUpdateOption) (*User, error) {
	u.setScope("users", uid)

	var info userInfo

	info.Name = opt.Name
	info.Avatar = opt.Avatar
	info.Link = opt.Link
	info.Role = opt.Role
	if opt.Metadata != nil {
		info.Metadata = *opt.Metadata
	}
	if len(opt.Tags) > 0 {
		info.Tags = opt.Tags
	}
	if len(opt.Unset) > 0 {
		info.Unset = opt.Unset
	}

	body, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	req, err := buildRequest(u.setMethod(http.MethodPut).setBody(body))
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

func (u *UserScoped) Delete(uid string) error {
	u.setScope("users", uid)

	var req struct {
		Permanent bool `json:"permanent"`
	}
	req.Permanent = true
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	request, err := buildRequest(u.setMethod(http.MethodDelete).setBody(body))
	if err != nil {
		return err
	}
	var response struct {
		Data struct {
			ApiResult
		} `json:"data"`
	}
	err = doRequest(request, &response)
	if err != nil {
		return err
	}
	return nil
}

func (u *UserScoped) Get(uid string) (*User, error) {
	u.setScope("users", uid)

	req, err := buildRequest(u.setMethod(http.MethodGet))
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

type UserListOption struct {
	SearchKey       string
	SearchIn        []string
	Status          string
	Count           bool
	PerPage         int
	Page            int
	Role            string
	OnlyDeactivated bool
	WithDeactivated bool
}

func (u *UserScoped) List(opt UserListOption) ([]User, error) {
	req, err := buildRequest(u.setQueries(opt).setMethod(http.MethodGet))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []User `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (u *UserScoped) AuthToken(uid string) *AuthTokenScoped {
	u.setScope("users", uid)
	u.setScope("auth_tokens", "")

	return &AuthTokenScoped{chatRequest: u.chatRequest}
}
