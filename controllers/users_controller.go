package controllers

import (
	"errors"
	"net/http"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateUser(c *gin.Context) {
	createUser(c, models.InsertUser)
}

func NewCreateUserHandler(insertUser func(models.User) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		createUser(c, insertUser)
	}
}

func createUser(c *gin.Context, insertUser func(models.User) error) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := insertUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User inserted"})
}

func ListAllUsers(c *gin.Context) {
	users, err := models.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find users"})
		return
	} else {
		c.JSON(http.StatusOK, users)
	}
}

func FindUserByID(c *gin.Context) {

	userID := c.Param("id")
	user, err := models.FindByID(userID)

	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidUserID):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		case errors.Is(err, mongo.ErrNoDocuments):
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}
