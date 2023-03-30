package comet

import (
	"encoding/json"
	"net/http"
)

type AuthTokenScoped struct {
	*chatRequest
}

type AuthTokenCreateOption struct {
	Force bool `json:"force"`
}

type AuthToken struct {
	UID       string `json:"uid"`
	AuthToken string `json:"authToken"`
	CreateAt  int    `json:"createAt"`
}

func (a *AuthTokenScoped) Create(opt *AuthTokenCreateOption) (*AuthToken, error) {
	if opt != nil {
		body, err := json.Marshal(&opt)
		if err != nil {
			return nil, err
		}

		a.setBody(body)
	}

	req, err := buildRequest(a.setMethod(http.MethodPost))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data struct {
			AuthToken
		} `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return &response.Data.AuthToken, nil
}

func (a *AuthTokenScoped) Delete(token string) (*ApiResult, error) {
	a.setScope("auth_tokens", token)
	a.setMethod(http.MethodDelete)

	req, err := buildRequest(a)
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

func (a *AuthTokenScoped) List() ([]AuthToken, error) {
	req, err := buildRequest(a.setMethod(http.MethodGet))
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []AuthToken `json:"data"`
	}

	err = doRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}
