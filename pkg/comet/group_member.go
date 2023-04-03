package comet

import (
	"encoding/json"
	"net/http"
)

type GroupMemberScoped struct {
	*chatRequest
}

type groupMembersInfo struct {
	Admins       []string `json:"admins,omitempty"`
	Moderators   []string `json:"moderators,omitempty"`
	Participants []string `json:"participants,omitempty"`
	UsersToBan   []string `json:"usersToBan,omitempty"`
}

type GroupMemberOption struct {
	Admins       []string
	Moderators   []string
	Participants []string
	UsersToBan   []string
}

func (g *GroupMemberScoped) Add(opt GroupMemberOption) (*Group, error) {
	var info groupMembersInfo

	if len(opt.Admins) > 0 {
		info.Admins = opt.Admins
	}
	if len(opt.Moderators) > 0 {
		info.Moderators = opt.Moderators
	}
	if len(opt.Participants) > 0 {
		info.Participants = opt.Participants
	}
	if len(opt.UsersToBan) > 0 {
		info.UsersToBan = opt.UsersToBan
	}

	body, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}

	req, err := buildRequest(g.setMethod(http.MethodPost).setBody(body))
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

func (g *GroupMemberScoped) Kick(uid string) (*ApiResult, error) {
	g.setScope("members", uid)
	g.setMethod(http.MethodDelete)

	req, err := buildRequest(g)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data struct {
			ApiResult
		} `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return &response.Data.ApiResult, nil
}

type GroupMemberListOption struct {
	PerPage int
	Page    int
	Scopes  []string
}

func (g *GroupMemberScoped) List(opt GroupMemberOption) ([]User, error) {
	req, err := buildRequest(g.setQueries(opt).setMethod(http.MethodGet))
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
