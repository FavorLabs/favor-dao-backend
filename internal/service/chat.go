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

func formatValidUrl(s string) string {
	return fmt.Sprintf("http://%s", strings.TrimPrefix(s, "http://"))
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

func GetGroupID(daoId string) string {
	return groupId(daoId)
}

func NetworkTag() string {
	return fmt.Sprintf("net_%d", conf.ExternalAppSetting.NetworkID)
}

func RegionTag() string {
	return fmt.Sprintf("region_%s", conf.ExternalAppSetting.Region)
}

func CreateChatUser(ctx context.Context, address, name, avatar string) error {
	uid := userId(address)

	_, err := chat.Scoped().Context(ctx).Users().Get(uid)
	if err != nil {
		switch e := err.(type) {
		case comet.RestApiError:
			if e.Inner.Code == "ERR_UID_NOT_FOUND" {
				_, err := chat.Scoped().Context(ctx).Users().Create(uid, name, &comet.UserCreateOption{
					Tags:        []string{RegionTag(), NetworkTag()},
					Avatar:      formatValidUrl(avatar),
					ReturnToken: false,
				})
				if err != nil {
					return err
				}

				return nil
			}
		}

		return err
	}

	return nil
}

func UpdateChatUser(ctx context.Context, address, name, avatar, token string) error {
	uid := userId(address)

	_, err := chat.Scoped().Context(ctx).Users().Update(uid, comet.UserUpdateOption{
		Tags:   []string{RegionTag(), NetworkTag()},
		Name:   name,
		Avatar: formatValidUrl(avatar),
		Token:  token,
	})
	if err != nil {
		return err
	}

	return nil
}

func DeleteChatUser(ctx context.Context, address string) error {
	uid := userId(address)

	err := chat.Scoped().Context(ctx).Users().Delete(uid)
	if err != nil {
		switch e := err.(type) {
		case comet.RestApiError:
			if e.Inner.Code == "ERR_UID_NOT_FOUND" {
				return nil
			}
		}
		return err
	}

	return nil
}

func GetAuthToken(ctx context.Context, address string) (string, error) {
	uid := userId(address)

	tokens, err := chat.Scoped().Context(ctx).Users().AuthToken(uid).List()
	if err != nil {
		return "", err
	}

	if len(tokens) == 0 {
		token, err := chat.Scoped().Context(ctx).Users().AuthToken(uid).Create(nil)
		if err != nil {
			return "", err
		}

		return token.AuthToken, nil
	}

	return tokens[0].AuthToken, nil
}

func CreateChatGroup(ctx context.Context, address, id, name, icon, desc string) (string, error) {
	uid := userId(address)
	gid := groupId(id)

	_, err := chat.Scoped().Context(ctx).Perform(uid).Groups().Create(gid, name, comet.PublicGroup, &comet.GroupCreateOption{
		Owner: address,
		Icon:  formatValidUrl(icon),
		Desc:  desc,
		Tags: []string{
			RegionTag(),
			NetworkTag(),
			fmt.Sprintf("DAO%s", name),
		},
	})
	if err != nil {
		return gid, err
	}

	return gid, nil
}

func UpdateChatGroup(ctx context.Context, address, id, name, icon, desc string) error {
	uid := userId(address)
	gid := groupId(id)

	_, err := chat.Scoped().Context(ctx).Perform(uid).Groups().Update(gid, comet.GroupUpdateOption{
		Name: name,
		Icon: formatValidUrl(icon),
		Desc: desc,
		Tags: []string{
			RegionTag(),
			NetworkTag(),
			fmt.Sprintf("DAO%s", name),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func DeleteGroup(ctx context.Context, daoId string) (err error) {
	gid := groupId(daoId)
	_, err = chat.Scoped().Context(ctx).Groups().Delete(gid)
	if err != nil {
		switch e := err.(type) {
		case comet.RestApiError:
			if e.Inner.Code == "ERR_GUID_NOT_FOUND" {
				return nil
			}
		}
		return
	}
	return
}

func KickGroupMembers(ctx context.Context, daoId, address string) (gid string, err error) {
	uid := userId(address)
	gid = groupId(daoId)
	_, err = chat.Scoped().Context(ctx).Groups().Members(gid).Kick(uid)
	if err != nil {
		switch e := err.(type) {
		case comet.RestApiError:
			if e.Inner.Code == "ERR_UID_NOT_FOUND" || e.Inner.Code == "ERR_GUID_NOT_FOUND" {
				err = nil
				return
			}
		}
		return
	}
	return
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

func ListChatGroups(daoId string, page, perPage int) ([]comet.Group, error) {
	dao, err := GetDao(daoId)
	if err != nil {
		return nil, err
	}

	uid := userId(dao.Address)

	// TODO make sure return sames with logged list in database
	return chat.Scoped().Perform(uid).Groups().List(comet.GroupListOption{
		Tags:      []string{RegionTag(), NetworkTag(), fmt.Sprintf("DAO%s", dao.Name)},
		Type:      "public",
		HasJoined: true,
		SortBy:    "createdAt",
		SortOrder: "desc",
		Page:      page,
		PerPage:   perPage,
	})
}
