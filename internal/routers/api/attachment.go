package api

import (
	"favor-dao-backend/pkg/app"
	"favor-dao-backend/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func GeneratePath(s string) string {
	n := len(s)
	if n <= 2 {
		return s
	}

	return GeneratePath(s[:n-2]) + "/" + s[n-2:]
}

func GetFileExt(s string) (string, error) {
	switch s {
	case "image/png":
		return ".png", nil
	case "image/jpg":
		return ".jpg", nil
	case "image/jpeg":
		return ".jpeg", nil
	case "image/gif":
		return ".gif", nil
	case "video/mp4":
		return ".mp4", nil
	case "video/quicktime":
		return ".mov", nil
	case "application/zip":
		return ".zip", nil
	default:
		return "", errcode.FileInvalidExt.WithDetails("only png/jpg/gif/mp4/mov/zip")
	}
}

func fileCheck(uploadType string, size int64) error {
	if uploadType != "public/video" &&
		uploadType != "public/image" &&
		uploadType != "public/avatar" &&
		uploadType != "attachment" {
		return errcode.InvalidParams
	}

	if size > 1024*1024*100 {
		return errcode.FileInvalidSize.WithDetails("Maximum allowed 100MB")
	}

	return nil
}

func UploadAttachment(c *gin.Context) {
	response := app.NewResponse(c)

	uploadType := c.Request.FormValue("type")
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		logrus.Errorf("api.UploadAttachment err: %v", err)
		response.ToErrorResponse(errcode.FileUploadFailed)
		return
	}
	defer file.Close()

	if err = fileCheck(uploadType, fileHeader.Size); err != nil {
		cErr, _ := err.(*errcode.Error)
		response.ToErrorResponse(cErr)
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	fileExt, err := GetFileExt(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		logrus.Errorf("GetFileExt err: %v", err)
		response.ToErrorResponse(err.(*errcode.Error))
		return
	}

	// Generate random paths
	randomPath := uuid.Must(uuid.NewV4()).String()
	ossSavePath := uploadType + "/" + GeneratePath(randomPath[:8]) + "/" + randomPath[9:] + fileExt

	objectUrl, err := objectStorage.PutObject(ossSavePath, file, fileHeader.Size, contentType, false)
	if err != nil {
		logrus.Errorf("putObject err: %v", err)
		response.ToErrorResponse(errcode.FileUploadFailed)
		return
	}

	response.ToResponse(objectUrl)
}
