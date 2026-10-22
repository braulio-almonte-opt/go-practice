package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/braulio-almonte-opt/go-practice/repository"
	"github.com/braulio-almonte-opt/go-practice/service"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	r := gin.Default()
	client, err := models.ConnectDatabase()
	if err != nil {
		return err
	}
	var userRepo repository.UserRepository = repository.NewMongoUserDAO(
		client,
		models.DatabaseName,
		models.UserCollectionName,
	)
	userService := service.NewUserService(userRepo)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("disconnect MongoDB: %v", err)
		}
	}()

	users := r.Group("/users")
	{
		//Create user
		users.POST("", controllers.NewCreateUserHandler(userService.CreateUser))

		//Get user by JWT
		// users.GET("/users/me", controllers.GetUserByJWT)

		//Get user by ID
		users.GET("/:id", controllers.NewFindUserByIDHandler(userService.FindByID))

		//Get all users
		users.GET("", controllers.NewListAllUsersHandler(userService.ListUsers))
	}
	log.Println("Server started")
	if err := r.Run(); err != nil {
		return err
	}
	return nil
}
