package test

import (
	"context"

	"github.com/braulio-almonte-opt/go-practice/auth"
	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/braulio-almonte-opt/go-practice/service"
	"github.com/gin-gonic/gin"
)

func setupProtectedUsersRouter(
	jwtManager *auth.JWTManager,
	findByID func(context.Context, string) (models.User, error),
	findAll func(context.Context) ([]models.User, error),
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	users := router.Group("/users", jwtManager.Middleware())
	users.GET("/me", controllers.NewCurrentUserHandler(findByID))
	users.GET(
		"/:id",
		auth.RequireOwnerOrAdmin(),
		controllers.NewFindUserByIDHandler(findByID),
	)
	users.GET(
		"",
		auth.RequireAdmin(),
		controllers.NewListAllUsersHandler(findAll),
	)
	return router
}

func setupLoginRouter(login func(context.Context, string, string) (service.LoginResult, error)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", controllers.NewLoginHandler(login))
	return router
}

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
