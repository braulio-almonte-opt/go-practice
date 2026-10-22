package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/braulio-almonte-opt/go-practice/models"
	repositorymocks "github.com/braulio-almonte-opt/go-practice/repository/mocks"
	"go.mongodb.org/mongo-driver/mongo"
)

type stubTokenIssuer struct {
	token     string
	expiresAt time.Time
	err       error
	user      models.User
}

func (s *stubTokenIssuer) Issue(user models.User) (string, time.Time, error) {
	s.user = user
	return s.token, s.expiresAt, s.err
}

func TestUserServiceDelegatesToRepository(t *testing.T) {
	expectedUser := models.User{UserName: "jane"}
	expectedUsers := []models.User{expectedUser}
	repo := new(repositorymocks.MockUserRepository)
	userService := NewUserService(repo, &stubTokenIssuer{})
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
	userService := NewUserService(repo, &stubTokenIssuer{})
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
			userService := NewUserService(repo, &stubTokenIssuer{})

			err := test.call(userService, ctx)
			if !errors.Is(err, test.err) {
				t.Errorf("lookup error = %v, want an error matching %v", err, test.err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserServiceLoginIssuesTokenForValidCredentials(t *testing.T) {
	ctx := context.Background()
	user := models.User{
		UserName: "jane",
		Password: "correct-password",
	}
	expiresAt := time.Now().Add(time.Hour)
	issuer := &stubTokenIssuer{token: "signed-token", expiresAt: expiresAt}
	repo := new(repositorymocks.MockUserRepository)
	repo.EXPECT().FindByUsername(ctx, user.UserName).Return(user, nil).Once()

	result, err := NewUserService(repo, issuer).Login(ctx, user.UserName, user.Password)
	if err != nil {
		t.Fatalf("Login returned an error: %v", err)
	}
	if result.AccessToken != issuer.token || !result.ExpiresAt.Equal(expiresAt) {
		t.Errorf("Login returned %#v, want token %q expiring at %v", result, issuer.token, expiresAt)
	}
	if issuer.user != user {
		t.Errorf("token issuer received user %#v, want %#v", issuer.user, user)
	}
	repo.AssertExpectations(t)
}

func TestUserServiceLoginRejectsInvalidCredentials(t *testing.T) {
	tests := []struct {
		name     string
		user     models.User
		repoErr  error
		password string
	}{
		{
			name:     "wrong password",
			user:     models.User{UserName: "jane", Password: "correct-password"},
			password: "wrong-password",
		},
		{
			name:     "unknown username",
			repoErr:  mongo.ErrNoDocuments,
			password: "any-password",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(repositorymocks.MockUserRepository)
			repo.EXPECT().
				FindByUsername(ctx, "jane").
				Return(test.user, test.repoErr).
				Once()
			issuer := &stubTokenIssuer{}

			_, err := NewUserService(repo, issuer).Login(ctx, "jane", test.password)
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Login error = %v, want %v", err, ErrInvalidCredentials)
			}
			if issuer.user != (models.User{}) {
				t.Errorf("token issuer must not run for invalid credentials; received %#v", issuer.user)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUserServiceLoginPropagatesRepositoryAndTokenErrors(t *testing.T) {
	ctx := context.Background()
	repositoryErr := errors.New("repository unavailable")
	repo := new(repositorymocks.MockUserRepository)
	repo.EXPECT().FindByUsername(ctx, "jane").Return(models.User{}, repositoryErr).Once()

	_, err := NewUserService(repo, &stubTokenIssuer{}).Login(ctx, "jane", "password")
	if !errors.Is(err, repositoryErr) {
		t.Errorf("Login repository error = %v, want it to match %v", err, repositoryErr)
	}
	repo.AssertExpectations(t)

	user := models.User{UserName: "jane", Password: "password"}
	tokenErr := errors.New("signing failed")
	repo = new(repositorymocks.MockUserRepository)
	repo.EXPECT().FindByUsername(ctx, "jane").Return(user, nil).Once()

	_, err = NewUserService(repo, &stubTokenIssuer{err: tokenErr}).Login(ctx, "jane", "password")
	if !errors.Is(err, tokenErr) {
		t.Errorf("Login token error = %v, want it to match %v", err, tokenErr)
	}
	repo.AssertExpectations(t)
}
