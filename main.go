package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/braulio-almonte-opt/go-practice/controllers"
	"github.com/braulio-almonte-opt/go-practice/models"
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
		users.POST("", controllers.NewCreateUserHandler(func(user models.User) error {
			return models.InsertUser(client, user)
		}))

		//Get user by JWT
		// users.GET("/users/me", controllers.GetUserByJWT)

		//Get user by ID
		users.GET("/:id", controllers.NewFindUserByIDHandler(func(userID string) (models.User, error) {
			return models.FindByID(client, userID)
		}))

		//Get all users
		users.GET("", controllers.NewListAllUsersHandler(func() ([]models.User, error) {
			return models.FindAll(client)
		}))
	}
	log.Println("Server started")
	if err := r.Run(); err != nil {
		return err
	}
	return nil
}
