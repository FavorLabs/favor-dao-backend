package comet

import (
	"encoding/json"
	"net/http"
)

type GroupScoped struct {
	*chatRequest
}

type GroupType string

const (
	PublicGroup   GroupType = "public"
	PasswordGroup           = "password"
	PrivateGroup            = "private"
)

type Group struct {
	GID                string    `json:"guid"`
	Name               string    `json:"name"`
	Type               GroupType `json:"type"`
	Icon               string    `json:"icon,omitempty"`
	Desc               string    `json:"description,omitempty"`
	Scope              string    `json:"scope,omitempty"`
	Owner              string    `json:"owner"`
	Tags               []string  `json:"tags,omitempty"`
	MembersCount       int       `json:"membersCount"`
	JoinedAt           int       `json:"joinedAt,omitempty"`
	HasJoined          bool      `json:"hasJoined,omitempty"`
	CreatedAt          int       `json:"createdAt"`
	UpdatedAt          int       `json:"updatedAt,omitempty"`
	ConversationId     string    `json:"conversationId"`
	OnlineMembersCount int       `json:"onlineMembersCount,omitempty"`
}

type groupInfo struct {
	GID      string            `json:"guid"`
	Name     string            `json:"name"`
	Type     GroupType         `json:"type"`
	Password string            `json:"password,omitempty"`
	Icon     string            `json:"icon,omitempty"`
	Desc     string            `json:"description,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Owner    string            `json:"owner,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Members  groupMembersInfo  `json:"members,omitempty"`
}

type GroupCreateOption struct {
	Password string
	Icon     string
	Desc     string
	Metadata map[string]string
	Owner    string
	Tags     []string
	Members  *groupMembersInfo
}

func (g *GroupScoped) Create(gid, name string, typ GroupType, opt *GroupCreateOption) (*Group, error) {
	info := groupInfo{
		GID:  gid,
		Name: name,
		Type: typ,
	}

	if opt != nil {
		info.Icon = opt.Icon
		info.Desc = opt.Desc
		if len(opt.Metadata) > 0 {
			info.Metadata = opt.Metadata
		}
		info.Owner = opt.Owner
		if len(opt.Tags) > 0 {
			info.Tags = opt.Tags
		}
		if opt.Members != nil {
			info.Members = *opt.Members
		}
	}

	body, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	req, err := buildRequest(g.setBody(body).setMethod(http.MethodPost))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data struct {
			Group
		} `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return &response.Data.Group, nil
}

type GroupListOption struct {
	SearchKey string
	SearchIn  []string
	SortBy    string
	SortOrder string
	PerPage   int
	Affix     string
	UpdatedAt int
	WithTags  bool
	Tags      []string
	Type      string
	Types     []string
	Page      int
	HasJoined bool
}

func (g *GroupScoped) List(opt GroupListOption) ([]Group, error) {
	req, err := buildRequest(g.setQueries(opt).setMethod(http.MethodGet))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []Group `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (g *GroupScoped) Get(gid string) (*Group, error) {
	g.setScope("groups", gid)

	req, err := buildRequest(g.setMethod(http.MethodGet))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data struct {
			Group
		} `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return &response.Data.Group, nil
}

func (g *GroupScoped) Members(gid string) *GroupMemberScoped {
	g.setScope("groups", gid)
	g.setScope("members", "")

	return &GroupMemberScoped{chatRequest: g.chatRequest}
}
