package model

import "go.mongodb.org/mongo-driver/mongo"

type PostAttachmentBill struct {
	*Model
	PostID     int64 `json:"post_id"`
	UserID     int64 `json:"user_id"`
	PaidAmount int64 `json:"paid_amount"`
}

func (p *PostAttachmentBill) Get(db *mongo.Database) (*PostAttachmentBill, error) {
	var pas PostAttachmentBill
	if p.Model != nil && p.ID > 0 {
		db = db.Where("id = ? AND is_del = ?", p.ID, 0)
	}
	if p.PostID > 0 {
		db = db.Where("post_id = ?", p.PostID)
	}
	if p.UserID > 0 {
		db = db.Where("user_id = ?", p.UserID)
	}

	err := db.First(&pas).Error
	if err != nil {
		return &pas, err
	}

	return &pas, nil
}

func (p *PostAttachmentBill) Create(db *mongo.Database) (*PostAttachmentBill, error) {
	err := db.Create(&p).Error

	return p, err
}
