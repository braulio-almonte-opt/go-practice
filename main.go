package main

import (
	"log"

	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	models.ConnectDatabase()

	users := r.Group("/users")
	{
		//Create user
		users.POST("", controllers.CreateUser)

		//Get user by JWT
		// users.GET("/users/me", controllers.GetUserByJWT)

		//Get user by ID
		users.GET("/:id", controllers.FindUserByID)

		//Get all users
		users.GET("", controllers.ListAllUsers)

	}
	log.Println("Server started")
	r.Run()
}
