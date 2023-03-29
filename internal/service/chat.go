package service

import (
	"context"
	"encoding/json"
	"favor-dao-backend/internal/conf"
	"favor-dao-backend/pkg/comet"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"io"
	"net/http"
	"strings"
)

type Session struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendly_name"`
	WalletAddr   string `json:"wallet_addr"`
}

func userId(address string) string {
	address = strings.TrimPrefix(address, "0x")

	return fmt.Sprintf("%d%s", conf.ExternalAppSetting.NetworkID, address)
}

func CreateChatUser(address, name, avatar string) (string, error) {
	cUid := userId(address)
	resp, err := chat.Scoped().Users().Create(cUid, name, &comet.UserCreateOption{
		//Avatar:      avatar,
		ReturnToken: true,
	})
	if err != nil {
		switch e := err.(type) {
		case comet.RestApiError:
			// if created
			if e.Inner.Code != "ERR_UID_ALREADY_EXISTS" {
				return "", err
			}
		default:
			return "", err
		}
	} else {
		return resp.AuthToken, nil
	}

	token, err := chat.Scoped().Users().AuthToken(cUid).Create(nil)
	if err != nil {
		return "", err
	}

	return token.AuthToken, nil
}

func groupId(name string) string {
	hashId := xxhash.Sum64String(fmt.Sprintf("group_%s", name))
	return fmt.Sprintf("%d%d", conf.ExternalAppSetting.NetworkID, hashId)
}

func CreateChatGroup(address, name, icon, desc string) (string, error) {
	cUid := userId(address)
	gid := groupId(name)
	_, err := chat.Scoped().Perform(cUid).Groups().Create(groupId(name), name, comet.PublicGroup, &comet.GroupCreateOption{
		Owner: address,
		Icon:  icon,
		Desc:  desc,
	})
	if err != nil {
		return gid, err
	}

	return gid, nil
}

func JoinOrLeaveGroup(ctx context.Context, daoName string, joinOrLeave bool, token string) (string, error) {
	groupId := groupId(daoName)

	url := fmt.Sprintf("https://%s.apiclient-%s.cometchat.io/v3/groups/%s/members", conf.ChatSetting.AppId, conf.ChatSetting.Region, groupId)

	var (
		req *http.Request
		err error
	)

	if joinOrLeave {
		// TODO join with password
		req, err = http.NewRequest(http.MethodPost, url, nil)
	} else {
		req, err = http.NewRequest(http.MethodDelete, url, nil)
	}

	if err != nil {
		return groupId, err
	}

	req.WithContext(ctx)
	req.Header.Set("authtoken", token)
	req.Header.Set("appid", conf.ChatSetting.AppId)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return groupId, err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode >= 300 {
		errBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return groupId, err
		}

		// parse restful error
		var restErr comet.RestApiError
		err = json.Unmarshal(errBody, &restErr)
		if err == nil {
			if joinOrLeave && restErr.Inner.Code == "ERR_ALREADY_JOINED" {
				return groupId, nil
			} else if !joinOrLeave && restErr.Inner.Code == "ERR_GROUP_NOT_JOINED" {
				return groupId, nil
			}

			return groupId, restErr
		}

		var apiErr comet.ApiError
		err = json.Unmarshal(errBody, &apiErr)
		if err == nil {
			return groupId, apiErr
		}

		return groupId, fmt.Errorf("operate group member(%d): %s", resp.StatusCode, string(errBody))
	}

	return groupId, nil
}
