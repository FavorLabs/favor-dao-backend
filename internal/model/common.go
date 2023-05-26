package model

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func findQuery(query []bson.M) bson.M {
	query = append(query, bson.M{"is_del": 0})
	if query != nil {
		if len(query) > 0 {
			return bson.M{"$and": query}
		}
	}
	return bson.M{}
}

func findQuery1(query []bson.M) bson.M {
	if query != nil {
		if len(query) > 0 {
			return bson.M{"$and": query}
		}
	}
	return bson.M{}
}

type PayStatus uint8

const (
	PaySubmit PayStatus = iota
	PaySuccess
	PayFailed
	PayRefund
)

// Model interface contains base methods that must be implemented by
// each model. If you're using the `DefaultModel` struct in your model,
// you don't need to implement any of these methods.
type Model interface {
	// PrepareID converts the id value if needed, then
	// returns it (e.g. convert string to objectId).
	PrepareID(id interface{}) (interface{}, error)

	GetID() interface{}
	SetID(id interface{})
}

// Collection interface bind specified table on database that
// must be implemented by each model.
type Collection interface {
	Model

	Table() string
}

// MarkDeleting interface defines a Mark method that marks
// a document as deleted in a MongoDB collection
type MarkDeleting interface {
	Mark(ctx context.Context) error
}

// CollectionMarkDeleting is an interface that combines the Collection
// and MarkDeleting interfaces. It represents a MongoDB collection
// that supports marking documents as deleted.
type CollectionMarkDeleting interface {
	Collection
	MarkDeleting
}

// IDField struct contains a model's ID field.
type IDField struct {
	ID primitive.ObjectID `json:"id" bson:"_id,omitempty"`
}

func (f *IDField) PrepareID(id interface{}) (interface{}, error) {
	if idStr, ok := id.(string); ok {
		return primitive.ObjectIDFromHex(idStr)
	}

	// Otherwise id must be ObjectId
	return id, nil
}

// GetID method returns a model's ID
func (f *IDField) GetID() interface{} {
	return f.ID
}

// SetID sets the value of a model's ID field.
func (f *IDField) SetID(id interface{}) {
	f.ID = id.(primitive.ObjectID)
}

// DateFields struct contains the `created_at` and `updated_at`
// fields that autofill when inserting or updating a model.
type DateFields struct {
	CreatedAt int64 `json:"created_on" bson:"created_on"`
	UpdatedAt int64 `json:"modified_on" bson:"modified_on"`
}

// Creating hook is used here to set the `created_at` field
// value when inserting a new model into the database.
func (f *DateFields) Creating(_ context.Context) error {
	f.CreatedAt = time.Now().Unix()
	return nil
}

// Saving hook is used here to set the `updated_at` field
// value when creating or updating a model.
func (f *DateFields) Saving(_ context.Context) error {
	f.UpdatedAt = time.Now().Unix()
	return nil
}

// SoftDeleteField struct contains the `deleted_at` field that autofill
// when marking delete a model
type SoftDeleteField struct {
	DeletedAt int64 `json:"-" bson:"deleted_on,omitempty"`
}

func (d *SoftDeleteField) Mark(_ context.Context) error {
	d.DeletedAt = time.Now().Unix()
	return nil
}

const (
	ID = "_id"

	CreatedAtField = "created_on"
	UpdatedAtField = "modified_on"
)

type DefaultModel struct {
	IDField    `bson:",inline"`
	DateFields `bson:",inline"`
}

func (m *DefaultModel) Creating(ctx context.Context) error {
	return m.DateFields.Creating(ctx)
}

func (m *DefaultModel) Saving(ctx context.Context) error {
	return m.DateFields.Saving(ctx)
}
