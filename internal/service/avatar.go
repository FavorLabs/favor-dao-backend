package service

import (
	"math/rand"
	"time"
)

var defaultAvatars = []string{
	"default_avatar_zoe",
	"default_avatar_william",
	"default_avatar_walter",
	"default_avatar_thomas",
	"default_avatar_taylor",
	"default_avatar_sophia",
	"default_avatar_sam",
	"default_avatar_ryan",
	"default_avatar_ruby",
	"default_avatar_quinn",
	"default_avatar_paul",
	"default_avatar_owen",
	"default_avatar_olivia",
	"default_avatar_norman",
	"default_avatar_nora",
	"default_avatar_natalie",
	"default_avatar_naomi",
	"default_avatar_miley",
	"default_avatar_mike",
	"default_avatar_lucas",
	"default_avatar_kylie",
	"default_avatar_julia",
	"default_avatar_joshua",
	"default_avatar_john",
	"default_avatar_jane",
	"default_avatar_jackson",
	"default_avatar_ivy",
	"default_avatar_isaac",
	"default_avatar_henry",
	"default_avatar_harry",
	"default_avatar_harold",
	"default_avatar_hanna",
	"default_avatar_grace",
	"default_avatar_george",
	"default_avatar_freddy",
	"default_avatar_frank",
	"default_avatar_finn",
	"default_avatar_emma",
	"default_avatar_emily",
	"default_avatar_edward",
	"default_avatar_clara",
	"default_avatar_claire",
	"default_avatar_chloe",
	"default_avatar_audrey",
	"default_avatar_arthur",
	"default_avatar_anna",
	"default_avatar_andy",
	"default_avatar_alfred",
	"default_avatar_alexa",
	"default_avatar_abigail",
}

func GetRandomAvatar() string {
	rand.Seed(time.Now().UnixMicro())
	return defaultAvatars[rand.Intn(len(defaultAvatars))]
}
