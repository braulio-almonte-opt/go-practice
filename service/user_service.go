package service

import (
	"context"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/braulio-almonte-opt/go-practice/repository"
)

type UserService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) *UserService {
	return &UserService{users: users}
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
