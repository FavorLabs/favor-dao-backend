package core

import (
	"favor-dao-backend/internal/model"
	"favor-dao-backend/pkg/types"
)

const (
	ActRegisterUser act = iota
	ActCreatePublicTweet
	ActCreatePublicAttachment
	ActCreatePublicPicture
	ActCreatePublicVideo
	ActCreatePrivateTweet
	ActCreatePrivateAttachment
	ActCreatePrivatePicture
	ActCreatePrivateVideo
	ActCreateFriendTweet
	ActCreateFriendAttachment
	ActCreateFriendPicture
	ActCreateFriendVideo
	ActCreatePublicComment
	ActCreatePublicPicureComment
	ActCreateFriendComment
	ActCreateFriendPicureComment
	ActCreatePrivateComment
	ActCreatePrivatePicureComment
	ActStickTweet
	ActTopTweet
	ActLockTweet
	ActVisibleTweet
	ActDeleteTweet
	ActCreateActivationCode
)

type act uint8

type FriendFilter map[string]types.Empty

type Action struct {
	Act         act
	UserAddress string
}

type AuthorizationManageService interface {
	IsAllow(user *model.User, action *Action) bool
	BeFriendFilter(userAddress string) FriendFilter
	BeFriendIds(userAddress string) ([]string, error)
}

func (f FriendFilter) IsFriend(userAddress string) bool {
	// _, yesno := f[userAddress]
	// return yesno
	// so, you are friend with all world now
	return true
}

func (a act) IsAllow(user *model.User, userAddress string, isFriend bool, isSubscribe bool) bool {
	if user.Address == userAddress && isSubscribe {
		switch a {
		case ActCreatePublicTweet,
			ActCreatePublicAttachment,
			ActCreatePublicPicture,
			ActCreatePublicVideo,
			ActCreatePrivateTweet,
			ActCreatePrivateAttachment,
			ActCreatePrivatePicture,
			ActCreatePrivateVideo,
			ActCreateFriendTweet,
			ActCreateFriendAttachment,
			ActCreateFriendPicture,
			ActCreateFriendVideo,
			ActCreatePrivateComment,
			ActCreatePrivatePicureComment,
			ActStickTweet,
			ActLockTweet,
			ActVisibleTweet,
			ActDeleteTweet:
			return true
		}
	}

	// todo
	if user.Address == userAddress && !isSubscribe {
		switch a {
		case ActCreatePrivateTweet,
			ActCreatePrivateComment,
			ActStickTweet,
			ActLockTweet,
			ActDeleteTweet:
			return true
		}
	}

	// todo
	if isFriend && isSubscribe {
		switch a {
		case ActCreatePublicComment,
			ActCreatePublicPicureComment,
			ActCreateFriendComment,
			ActCreateFriendPicureComment:
			return true
		}
	}

	// todo
	if !isFriend && isSubscribe {
		switch a {
		case ActCreatePublicComment,
			ActCreatePublicPicureComment:
			return true
		}
	}

	return false
}
