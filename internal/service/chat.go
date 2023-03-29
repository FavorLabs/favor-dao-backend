package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/pkg/comet"
	"github.com/cespare/xxhash/v2"
)

type Session struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendly_name"`
	WalletAddr   string `json:"wallet_addr"`
}

func genId(id string) string {
	return strconv.FormatUint(
		xxhash.Sum64String(fmt.Sprintf("%s-%d-%s", conf.ExternalAppSetting.Region, conf.ExternalAppSetting.NetworkID, id)),
		10,
	)
}

func userId(address string) string {
	return genId(strings.TrimPrefix(address, "0x"))
}

func groupId(id string) string {
	return genId(fmt.Sprintf("group_%s", id))
}

func networkTag() string {
	return fmt.Sprintf("net_%d", conf.ExternalAppSetting.NetworkID)
}

func regionTag() string {
	return fmt.Sprintf("region_%s", conf.ExternalAppSetting.Region)
}

func CreateChatUser(address, name, avatar string) (string, error) {
	cUid := userId(address)
	resp, err := chat.Scoped().Users().Create(cUid, name, &comet.UserCreateOption{
		Tags:        []string{regionTag(), networkTag()},
		Avatar:      fmt.Sprintf("http://%s", strings.TrimPrefix(avatar, "http://")),
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

func CreateChatGroup(address, id, name, icon, desc string) (string, error) {
	uid := userId(address)
	gid := groupId(id)

	_, err := chat.Scoped().Perform(uid).Groups().Create(gid, name, comet.PublicGroup, &comet.GroupCreateOption{
		Owner: address,
		Icon:  icon,
		Desc:  desc,
		Tags: []string{
			regionTag(),
			networkTag(),
			fmt.Sprintf("DAO%s", name)},
	})
	if err != nil {
		return gid, err
	}

	return gid, nil
}

func JoinOrLeaveGroup(ctx context.Context, daoId string, joinOrLeave bool, token string) (string, error) {
	groupId := groupId(daoId)

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

func ListChatGroups(address, name string, page, perPage int) ([]comet.Group, error) {
	uid := userId(address)

	// TODO make sure return sames with logged list in database
	return chat.Scoped().Perform(uid).Groups().List(comet.GroupListOption{
		Tags:      []string{networkTag(), fmt.Sprintf("DAO%s", name)},
		HasJoined: true,
		SortBy:    "createdAt",
		SortOrder: "desc",
		Page:      page,
		PerPage:   perPage,
	})
}
