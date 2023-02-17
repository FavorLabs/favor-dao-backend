package core

type AttachmentCheckService interface {
	CheckAttachment(uri string) error
}
