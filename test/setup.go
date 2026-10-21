package test

import (
	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
)

func setupRouter(insertUser func(models.User) error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/users", controllers.NewCreateUserHandler(insertUser))
	return router
}
