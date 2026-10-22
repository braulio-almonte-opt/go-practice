//go:build wireinject

package main

import (
	"github.com/braulio-almonte-opt/go-practice/auth"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/braulio-almonte-opt/go-practice/repository"
	"github.com/braulio-almonte-opt/go-practice/service"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"go.mongodb.org/mongo-driver/mongo"
)

//go:generate go run github.com/google/wire/cmd/wire@v0.7.0

func provideUserRepository(client *mongo.Client) repository.UserRepository {
	return repository.NewMongoUserDAO(
		client,
		models.DatabaseName,
		models.UserCollectionName,
	)
}

func provideJWTManager(secret []byte) (*auth.JWTManager, error) {
	return auth.NewJWTManager(secret)
}

var userApplicationSet = wire.NewSet(
	provideUserRepository,
	provideJWTManager,
	wire.Bind(new(service.TokenIssuer), new(*auth.JWTManager)),
	service.NewUserService,
	newRouter,
)

func initializeRouter(client *mongo.Client, jwtSecret []byte) (*gin.Engine, error) {
	wire.Build(userApplicationSet)
	return nil, nil
}
