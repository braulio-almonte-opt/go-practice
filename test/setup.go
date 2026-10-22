package test

import (
	"context"

	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
)

func setupRouter(insertUser func(context.Context, models.User) error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/users", controllers.NewCreateUserHandler(insertUser))
	return router
}

func setupListUsersRouter(findAll func(context.Context) ([]models.User, error)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/users", controllers.NewListAllUsersHandler(findAll))
	return router
}

func setupFindUserRouter(findByID func(context.Context, string) (models.User, error)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/users/:id", controllers.NewFindUserByIDHandler(findByID))
	return router
}
