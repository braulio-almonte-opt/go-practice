package controllers

import (
	"net/http"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
)

func CreateUser(c *gin.Context) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error:": err.Error()})
		return
	}

	err := models.InsertUser(user)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error:": "Failed to insert user"})

	}

	c.JSON(http.StatusCreated, gin.H{"message": "User inserted"})

}

func ListAllUsers(c *gin.Context) {
	users := models.FindAll()
	c.JSON(http.StatusOK, users)
}

func FindUserByID(c *gin.Context) {

	userID := c.Param("id")
	user, err := models.FindByID(userID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Failed to find user"})
	} else {
		c.JSON(http.StatusOK, user)
	}
}
