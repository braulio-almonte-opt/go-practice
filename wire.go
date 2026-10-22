//go:build wireinject

package main

import (
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

var userApplicationSet = wire.NewSet(
	provideUserRepository,
	service.NewUserService,
	newRouter,
)

func initializeRouter(client *mongo.Client) *gin.Engine {
	wire.Build(userApplicationSet)
	return nil
}
