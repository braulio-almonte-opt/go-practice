package main

import (
	"github.com/braulio-almonte-opt/go-practice/auth"
	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/service"
	"github.com/gin-gonic/gin"
)

func newRouter(userService *service.UserService, jwtManager *auth.JWTManager) *gin.Engine {
	r := gin.Default()
	r.POST("/login", controllers.NewLoginHandler(userService.Login))
	users := r.Group("/users")
	{
		users.POST("", controllers.NewCreateUserHandler(userService.CreateUser))
	}
	protectedUsers := r.Group("/users", jwtManager.Middleware())
	{
		protectedUsers.GET("/me", controllers.NewCurrentUserHandler(userService.FindByID))
		protectedUsers.GET("/:id", controllers.NewFindUserByIDHandler(userService.FindByID))
		protectedUsers.GET("", controllers.NewListAllUsersHandler(userService.ListUsers))
	}
	return r
}
