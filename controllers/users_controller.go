package controllers

import (
	"errors"
	"net/http"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type userResponse struct {
	ID       primitive.ObjectID `json:"_id,omitempty"`
	UserName string             `json:"username"`
	RealName string             `json:"realname"`
	LastName string             `json:"lastname"`
	Balance  float64            `json:"balance"`
	Role     string             `json:"role"`
}

func newUserResponse(user models.User) userResponse {
	return userResponse{
		ID:       user.ID,
		UserName: user.UserName,
		RealName: user.RealName,
		LastName: user.LastName,
		Balance:  user.Balance,
		Role:     user.Role,
	}
}

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
	listAllUsers(c, models.FindAll)
}

func NewListAllUsersHandler(findAll func() ([]models.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		listAllUsers(c, findAll)
	}
}

func listAllUsers(c *gin.Context, findAll func() ([]models.User, error)) {
	users, err := findAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find users"})
		return
	}

	response := make([]userResponse, len(users))
	for i, user := range users {
		response[i] = newUserResponse(user)
	}
	c.JSON(http.StatusOK, response)
}

func FindUserByID(c *gin.Context) {
	findUserByID(c, models.FindByID)
}

func NewFindUserByIDHandler(findByID func(string) (models.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		findUserByID(c, findByID)
	}
}

func findUserByID(c *gin.Context, findByID func(string) (models.User, error)) {
	user, err := findByID(c.Param("id"))
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

	c.JSON(http.StatusOK, newUserResponse(user))
}
