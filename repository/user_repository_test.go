package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/braulio-almonte-opt/go-practice/models"
)

func TestFindByIDRejectsInvalidObjectID(t *testing.T) {
	dao := &MongoUserDAO{}

	_, err := dao.FindByID(context.Background(), "not-an-object-id")
	if !errors.Is(err, models.ErrInvalidUserID) {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}
