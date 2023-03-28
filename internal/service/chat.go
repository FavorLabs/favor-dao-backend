package service

import (
	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/model"
	chatModel "favor-dao-backend/internal/model/chat"
	"favor-dao-backend/pkg/comet"
	"fmt"
	"github.com/cespare/xxhash/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"net/http"
)

type Session struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendly_name"`
	WalletAddr   string `json:"wallet_addr"`
}

func userId(uid string) string {
	return fmt.Sprintf("%d%s", conf.ExternalAppSetting.NetworkID, uid)
}

func CreateChatUser(uid, name, avatar string) (string, error) {
	cUid := userId(uid)
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

func CreateChatGroup(uidHex primitive.ObjectID, daoId, name, icon, desc string) error {
	uid := uidHex.Hex()
	cUid := userId(uid)
	group, err := chat.Scoped().Perform(cUid).Groups().Create(groupId(name), name, comet.PublicGroup, &comet.GroupCreateOption{
		Owner: uid,
		Icon:  icon,
		Desc:  desc,
	})
	if err != nil {
		return err
	}

	daoIdHex, _ := primitive.ObjectIDFromHex(daoId)
	_, err = ds.LinkDao(&chatModel.Group{
		GroupID: group.GID,
		DaoID:   daoIdHex,
		OwnerID: cUid,
	})
	if err != nil {
		return err
	}

	return nil
}

func JoinOrLeaveGroup(uid primitive.ObjectID, daoId string, joinOrLeave bool, token string) error {
	id, err := primitive.ObjectIDFromHex(daoId)
	if err != nil {
		return err
	}
	dao, err := ds.GetDao(&model.Dao{
		ID: id,
	})
	if err != nil {
		return err
	}

	groupId := groupId(dao.Name)

	/// Client API start =========
	url := fmt.Sprintf("https://%s.apiclient-%s.cometchat.io/v3/groups/%s/members", conf.ChatSetting.AppId, conf.ChatSetting.Region, groupId)

	var req *http.Request

	if joinOrLeave {
		// TODO join with password
		req, err = http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return err
		}
	} else {
		req, err = http.NewRequest(http.MethodDelete, url, nil)
	}

	req.Header.Set("authtoken", token)
	req.Header.Set("appid", conf.ChatSetting.AppId)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode >= 300 {
		errBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("operate group member(%d): %s", resp.StatusCode, string(errBody))
	}
	/// Client API end ===========

	// link or unlink
	if joinOrLeave {
		_, err = ds.LinkDao(&chatModel.Group{
			DaoID:   id,
			OwnerID: userId(uid.Hex()),
			GroupID: groupId,
		})
	} else {
		groupLink, err := ds.FindGroupByDao(daoId)
		if err != nil {
			return err
		}

		err = ds.UnlinkDao(groupLink)
	}

	return err
}
