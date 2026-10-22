package models

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ErrInvalidUserID = errors.New("invalid user ID")

type User struct {
	ID       primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	UserName string             `json:"username"`
	Password string             `json:"password"`
	RealName string             `json:"realname"`
	LastName string             `json:"lastname"`
	Balance  float64            `json:"balance"`
	Role     string             `json:"role"`
}
