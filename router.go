package main

import (
	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/service"
	"github.com/gin-gonic/gin"
)

func newRouter(userService *service.UserService) *gin.Engine {
	r := gin.Default()
	r.POST("/login", controllers.NewLoginHandler(userService.Login))
	users := r.Group("/users")
	{
		users.POST("", controllers.NewCreateUserHandler(userService.CreateUser))
		users.GET("/:id", controllers.NewFindUserByIDHandler(userService.FindByID))
		users.GET("", controllers.NewListAllUsersHandler(userService.ListUsers))
	}
	return r
}
