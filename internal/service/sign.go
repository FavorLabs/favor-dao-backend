package service

import (
	"fmt"
	"sort"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/pkg/util"
	"github.com/sirupsen/logrus"
)

func GetParamSign(param map[string]interface{}, secretKey string) string {
	signRaw := ""

	rawStrs := []string{}
	for k, v := range param {
		if k != "sign" {
			rawStrs = append(rawStrs, k+"="+fmt.Sprintf("%v", v))
		}
	}

	sort.Strings(rawStrs)
	for _, v := range rawStrs {
		signRaw += v
	}

	if conf.ServerSetting.RunMode == "debug" {
		logrus.Info(map[string]string{
			"signRaw": signRaw,
			"sysSign": util.EncodeMD5(signRaw + secretKey),
		})
	}

	return util.EncodeMD5(signRaw + secretKey)
}
