package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/braulio-almonte-opt/go-practice/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const testMongoURIEnv = "GO_PRACTICE_TEST_MONGODB_URI"

func TestMongoUserRepositoryCRUDIntegration(t *testing.T) {
	uri := os.Getenv(testMongoURIEnv)
	if uri == "" {
		t.Skipf("set %s to run MongoDB repository integration tests", testMongoURIEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect to test MongoDB: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := client.Disconnect(cleanupCtx); err != nil {
			t.Errorf("disconnect test MongoDB client: %v", err)
		}
	})
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		t.Fatalf("ping test MongoDB: %v", err)
	}

	databaseName := "go_practice_test_" + primitive.NewObjectID().Hex()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := client.Database(databaseName).Drop(cleanupCtx); err != nil {
			t.Errorf("drop isolated test database %q: %v", databaseName, err)
		}
	})

	repo := NewMongoUserDAO(client, databaseName, "users")
	expectedUser := models.User{
		UserName: "repository-test-user",
		Password: "test-password",
		RealName: "Repository",
		LastName: "Test",
		Balance:  42.5,
		Role:     "user",
	}
	if err := repo.InsertUser(ctx, expectedUser); err != nil {
		t.Fatalf("InsertUser returned an error: %v", err)
	}

	foundByUsername, err := repo.FindByUsername(ctx, expectedUser.UserName)
	if err != nil {
		t.Fatalf("FindByUsername returned an error: %v", err)
	}
	if foundByUsername.UserName != expectedUser.UserName ||
		foundByUsername.Password != expectedUser.Password ||
		foundByUsername.RealName != expectedUser.RealName ||
		foundByUsername.LastName != expectedUser.LastName ||
		foundByUsername.Balance != expectedUser.Balance ||
		foundByUsername.Role != expectedUser.Role {
		t.Errorf("FindByUsername returned %#v, want fields matching %#v", foundByUsername, expectedUser)
	}
	if foundByUsername.ID.IsZero() {
		t.Fatal("FindByUsername returned a user without a MongoDB ID")
	}

	foundByID, err := repo.FindByID(ctx, foundByUsername.ID.Hex())
	if err != nil {
		t.Fatalf("FindByID returned an error: %v", err)
	}
	if foundByID != foundByUsername {
		t.Errorf("FindByID returned %#v, want %#v", foundByID, foundByUsername)
	}

	users, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll returned an error: %v", err)
	}
	if len(users) != 1 || users[0] != foundByUsername {
		t.Errorf("FindAll returned %#v, want exactly %#v", users, foundByUsername)
	}

	_, err = repo.FindByUsername(ctx, "missing-repository-test-user")
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("FindByUsername missing-user error = %v, want %v", err, mongo.ErrNoDocuments)
	}

	_, err = repo.FindByID(ctx, primitive.NewObjectID().Hex())
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Errorf("FindByID missing-user error = %v, want %v", err, mongo.ErrNoDocuments)
	}
}

func TestMongoUserRepositoryReturnsErrorsForDisconnectedClient(t *testing.T) {
	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI("mongodb://127.0.0.1:27017"),
	)
	if err != nil {
		t.Fatalf("create MongoDB client: %v", err)
	}
	if err := client.Disconnect(context.Background()); err != nil {
		t.Fatalf("disconnect MongoDB client: %v", err)
	}

	repo := NewMongoUserDAO(client, "unused_test_database", "users")
	ctx := context.Background()

	if err := repo.InsertUser(ctx, models.User{UserName: "user"}); err == nil {
		t.Error("InsertUser returned no error for a disconnected client")
	}
	if _, err := repo.FindByUsername(ctx, "user"); err == nil {
		t.Error("FindByUsername returned no error for a disconnected client")
	}
	if _, err := repo.FindByID(ctx, primitive.NewObjectID().Hex()); err == nil {
		t.Error("FindByID returned no error for a disconnected client")
	}
	if users, err := repo.FindAll(ctx); err == nil {
		t.Errorf("FindAll returned users %#v without an error for a disconnected client", users)
	}
}
