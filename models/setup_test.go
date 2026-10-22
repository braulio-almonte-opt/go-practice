package models

import (
	"context"
	"testing"
	"time"
)

func TestConnectDatabaseReturnsErrorWhenMongoDBIsUnavailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	client, err := connectDatabase(ctx, "mongodb://127.0.0.1:1")
	if err == nil {
		if client != nil {
			_ = client.Disconnect(context.Background())
		}
		t.Fatal("expected an error when MongoDB is unavailable")
	}
	if client != nil {
		t.Fatal("expected no client when MongoDB ping fails")
	}
}
