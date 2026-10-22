package service

import (
	"context"
	"errors"
	"testing"

	"github.com/braulio-almonte-opt/go-practice/models"
)

type fakeUserRepository struct {
	ctx          context.Context
	insertedUser models.User
	username     string
	userID       string
	users        []models.User
	user         models.User
	err          error
}

func (f *fakeUserRepository) InsertUser(ctx context.Context, user models.User) error {
	f.ctx = ctx
	f.insertedUser = user
	return f.err
}

func (f *fakeUserRepository) FindByUsername(ctx context.Context, username string) (models.User, error) {
	f.ctx = ctx
	f.username = username
	return f.user, f.err
}

func (f *fakeUserRepository) FindByID(ctx context.Context, userID string) (models.User, error) {
	f.ctx = ctx
	f.userID = userID
	return f.user, f.err
}

func (f *fakeUserRepository) FindAll(ctx context.Context) ([]models.User, error) {
	f.ctx = ctx
	return f.users, f.err
}

func TestUserServiceDelegatesToRepository(t *testing.T) {
	expectedUser := models.User{UserName: "jane"}
	expectedUsers := []models.User{expectedUser}
	repo := &fakeUserRepository{user: expectedUser, users: expectedUsers}
	userService := NewUserService(repo)
	ctx := context.Background()

	if err := userService.CreateUser(ctx, expectedUser); err != nil {
		t.Fatalf("CreateUser returned unexpected error: %v", err)
	}
	if repo.ctx != ctx {
		t.Error("CreateUser did not pass the request context to the repository")
	}
	if repo.insertedUser != expectedUser {
		t.Errorf("CreateUser passed user %#v, want %#v", repo.insertedUser, expectedUser)
	}

	user, err := userService.FindByUsername(ctx, "jane")
	if err != nil {
		t.Fatalf("FindByUsername returned unexpected error: %v", err)
	}
	if repo.ctx != ctx {
		t.Error("FindByUsername did not pass the request context to the repository")
	}
	if repo.username != "jane" || user != expectedUser {
		t.Errorf("FindByUsername result = (%q, %#v), want (%q, %#v)", repo.username, user, "jane", expectedUser)
	}

	user, err = userService.FindByID(ctx, "user-id")
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if repo.ctx != ctx {
		t.Error("FindByID did not pass the request context to the repository")
	}
	if repo.userID != "user-id" || user != expectedUser {
		t.Errorf("FindByID result = (%q, %#v), want (%q, %#v)", repo.userID, user, "user-id", expectedUser)
	}

	users, err := userService.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers returned unexpected error: %v", err)
	}
	if repo.ctx != ctx {
		t.Error("ListUsers did not pass the request context to the repository")
	}
	if len(users) != len(expectedUsers) || users[0] != expectedUser {
		t.Errorf("ListUsers returned %#v, want %#v", users, expectedUsers)
	}
}

func TestUserServicePropagatesRepositoryErrors(t *testing.T) {
	repositoryErr := errors.New("repository unavailable")
	userService := NewUserService(&fakeUserRepository{err: repositoryErr})
	ctx := context.Background()
	user := models.User{UserName: "jane"}

	if err := userService.CreateUser(ctx, user); !errors.Is(err, repositoryErr) {
		t.Errorf("CreateUser error = %v, want %v", err, repositoryErr)
	}
	if _, err := userService.FindByUsername(ctx, "jane"); !errors.Is(err, repositoryErr) {
		t.Errorf("FindByUsername error = %v, want %v", err, repositoryErr)
	}
	if _, err := userService.FindByID(ctx, "user-id"); !errors.Is(err, repositoryErr) {
		t.Errorf("FindByID error = %v, want %v", err, repositoryErr)
	}
	if _, err := userService.ListUsers(ctx); !errors.Is(err, repositoryErr) {
		t.Errorf("ListUsers error = %v, want %v", err, repositoryErr)
	}
}
