package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/braulio-almonte-opt/go-practice/auth"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/braulio-almonte-opt/go-practice/repository"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type TokenIssuer interface {
	Issue(models.User) (string, time.Time, error)
}

type LoginResult struct {
	AccessToken string
	ExpiresAt   time.Time
}

type UserService struct {
	users       repository.UserRepository
	tokenIssuer TokenIssuer
}

func NewUserService(users repository.UserRepository, tokenIssuer TokenIssuer) *UserService {
	return &UserService{users: users, tokenIssuer: tokenIssuer}
}

func (s *UserService) CreateUser(ctx context.Context, user models.User) error {
	return s.users.InsertUser(ctx, user)
}

func (s *UserService) FindByUsername(ctx context.Context, username string) (models.User, error) {
	return s.users.FindByUsername(ctx, username)
}

func (s *UserService) FindByID(ctx context.Context, userID string) (models.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *UserService) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.users.FindAll(ctx)
}

func (s *UserService) Login(ctx context.Context, username, password string) (LoginResult, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, fmt.Errorf("find user for login: %w", err)
	}
	if user.Password != password {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, expiresAt, err := s.tokenIssuer.Issue(user)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue login token: %w", err)
	}
	return LoginResult{AccessToken: token, ExpiresAt: expiresAt}, nil
}

var _ TokenIssuer = (*auth.JWTManager)(nil)
