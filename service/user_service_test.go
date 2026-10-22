package service

import (
	"context"
	"errors"
	"testing"

	"github.com/braulio-almonte-opt/go-practice/models"
	repositorymocks "github.com/braulio-almonte-opt/go-practice/repository/mocks"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestUserServiceDelegatesToRepository(t *testing.T) {
	expectedUser := models.User{UserName: "jane"}
	expectedUsers := []models.User{expectedUser}
	repo := new(repositorymocks.MockUserRepository)
	userService := NewUserService(repo)
	ctx := context.Background()

	repo.EXPECT().InsertUser(ctx, expectedUser).Return(nil).Once()
	repo.EXPECT().FindByUsername(ctx, "jane").Return(expectedUser, nil).Once()
	repo.EXPECT().FindByID(ctx, "user-id").Return(expectedUser, nil).Once()
	repo.EXPECT().FindAll(ctx).Return(expectedUsers, nil).Once()

	if err := userService.CreateUser(ctx, expectedUser); err != nil {
		t.Fatalf("CreateUser returned unexpected error: %v", err)
	}

	user, err := userService.FindByUsername(ctx, "jane")
	if err != nil {
		t.Fatalf("FindByUsername returned unexpected error: %v", err)
	}
	if user != expectedUser {
		t.Errorf("FindByUsername returned %#v, want %#v", user, expectedUser)
	}

	user, err = userService.FindByID(ctx, "user-id")
	if err != nil {
		t.Fatalf("FindByID returned unexpected error: %v", err)
	}
	if user != expectedUser {
		t.Errorf("FindByID returned %#v, want %#v", user, expectedUser)
	}

	users, err := userService.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers returned unexpected error: %v", err)
	}
	if len(users) != len(expectedUsers) || users[0] != expectedUser {
		t.Errorf("ListUsers returned %#v, want %#v", users, expectedUsers)
	}

	repo.AssertExpectations(t)
}

func TestUserServicePropagatesRepositoryErrors(t *testing.T) {
	repositoryErr := errors.New("repository unavailable")
	repo := new(repositorymocks.MockUserRepository)
	userService := NewUserService(repo)
	ctx := context.Background()
	user := models.User{UserName: "jane"}

	repo.EXPECT().InsertUser(ctx, user).Return(repositoryErr).Once()
	repo.EXPECT().FindByUsername(ctx, "jane").Return(models.User{}, repositoryErr).Once()
	repo.EXPECT().FindByID(ctx, "user-id").Return(models.User{}, repositoryErr).Once()
	repo.EXPECT().FindAll(ctx).Return(nil, repositoryErr).Once()

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

	repo.AssertExpectations(t)
}

func TestUserServicePreservesLookupErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect func(*repositorymocks.MockUserRepository, context.Context)
		call   func(*UserService, context.Context) error
	}{
		{
			name: "invalid user ID",
			err:  models.ErrInvalidUserID,
			expect: func(repo *repositorymocks.MockUserRepository, ctx context.Context) {
				repo.EXPECT().FindByID(ctx, "invalid-id").Return(models.User{}, models.ErrInvalidUserID).Once()
			},
			call: func(userService *UserService, ctx context.Context) error {
				_, err := userService.FindByID(ctx, "invalid-id")
				return err
			},
		},
		{
			name: "user not found by ID",
			err:  mongo.ErrNoDocuments,
			expect: func(repo *repositorymocks.MockUserRepository, ctx context.Context) {
				repo.EXPECT().FindByID(ctx, "valid-id").Return(models.User{}, mongo.ErrNoDocuments).Once()
			},
			call: func(userService *UserService, ctx context.Context) error {
				_, err := userService.FindByID(ctx, "valid-id")
				return err
			},
		},
		{
			name: "user not found by username",
			err:  mongo.ErrNoDocuments,
			expect: func(repo *repositorymocks.MockUserRepository, ctx context.Context) {
				repo.EXPECT().FindByUsername(ctx, "missing").Return(models.User{}, mongo.ErrNoDocuments).Once()
			},
			call: func(userService *UserService, ctx context.Context) error {
				_, err := userService.FindByUsername(ctx, "missing")
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(repositorymocks.MockUserRepository)
			test.expect(repo, ctx)
			userService := NewUserService(repo)

			err := test.call(userService, ctx)
			if !errors.Is(err, test.err) {
				t.Errorf("lookup error = %v, want an error matching %v", err, test.err)
			}
			repo.AssertExpectations(t)
		})
	}
}
