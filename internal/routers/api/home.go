package api

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"image/color"
	"image/png"
	"time"

	"favor-dao-backend/internal/conf"
	"favor-dao-backend/internal/service"
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/debug"
	"favor-dao-backend/pkg/errcode"
	"favor-dao-backend/pkg/util"
	"github.com/afocus/captcha"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

//go:embed assets/comic.ttf
var comic []byte

func Version(c *gin.Context) {
	response := app.NewResponse(c)
	response.ToResponse(gin.H{
		"BuildInfo": debug.ReadBuildInfo(),
		"Settings": gin.H{
			"Bucket":     conf.ExternalAppSetting.UseBucket,
			"TagRegion":  service.RegionTag(),
			"TagNetwork": service.NetworkTag(),
			"Region":     conf.ExternalAppSetting.Region,
			"NetworkID":  conf.ExternalAppSetting.NetworkID,
		},
	})
}

func GetCaptcha(c *gin.Context) {
	cap := captcha.New()

	if err := cap.AddFontFromBytes(comic); err != nil {
		panic(err.Error())
	}

	cap.SetSize(160, 64)
	cap.SetDisturbance(captcha.MEDIUM)
	cap.SetFrontColor(color.RGBA{0, 0, 0, 255})
	cap.SetBkgColor(color.RGBA{218, 240, 228, 255})
	img, password := cap.Create(6, captcha.NUM)
	emptyBuff := bytes.NewBuffer(nil)
	_ = png.Encode(emptyBuff, img)

	key := util.EncodeMD5(uuid.Must(uuid.NewV4()).String())

	conf.Redis.Set(c, "DaoCaptcha:"+key, password, time.Minute*5)

	response := app.NewResponse(c)
	response.ToResponse(gin.H{
		"id":   key,
		"b64s": "data:image/png;base64," + base64.StdEncoding.EncodeToString(emptyBuff.Bytes()),
	})
}

func PostCaptcha(c *gin.Context) {
	param := service.PhoneCaptchaReq{}
	response := app.NewResponse(c)
	valid, errs := app.BindAndValid(c, &param)
	if !valid {
		logrus.Errorf("app.BindAndValid errs: %v", errs)
		response.ToErrorResponse(errcode.InvalidParams.WithDetails(errs.Errors()...))
		return
	}

	// Verify image verification code
	if res, err := conf.Redis.Get(c.Request.Context(), "DaoCaptcha:"+param.ImgCaptchaID).Result(); err != nil || res != param.ImgCaptcha {
		response.ToErrorResponse(errcode.ErrorCaptchaPassword)
		return
	}
	conf.Redis.Del(c.Request.Context(), "DaoCaptcha:"+param.ImgCaptchaID).Result()

	response.ToResponse(nil)
}
