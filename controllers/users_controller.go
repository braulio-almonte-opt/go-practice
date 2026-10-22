package controllers

import (
	"context"
	"errors"
	"net/http"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/braulio-almonte-opt/go-practice/service"
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

type loginRequest struct {
	UserName string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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

func NewLoginHandler(login func(context.Context, string, string) (service.LoginResult, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request loginRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
			return
		}

		result, err := login(c.Request.Context(), request.UserName, request.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token": result.AccessToken,
			"token_type":   "Bearer",
			"expires_at":   result.ExpiresAt,
		})
	}
}

func NewCreateUserHandler(insertUser func(context.Context, models.User) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		createUser(c, insertUser)
	}
}

func createUser(c *gin.Context, insertUser func(context.Context, models.User) error) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := insertUser(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User inserted"})
}

func NewListAllUsersHandler(findAll func(context.Context) ([]models.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		listAllUsers(c, findAll)
	}
}

func listAllUsers(c *gin.Context, findAll func(context.Context) ([]models.User, error)) {
	users, err := findAll(c.Request.Context())
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

func NewFindUserByIDHandler(findByID func(context.Context, string) (models.User, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		findUserByID(c, findByID)
	}
}

func findUserByID(c *gin.Context, findByID func(context.Context, string) (models.User, error)) {
	user, err := findByID(c.Request.Context(), c.Param("id"))
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
