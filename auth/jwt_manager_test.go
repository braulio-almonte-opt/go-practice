package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestJWTManagerIssuesSignedTokenWithIdentityAndRole(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	manager, err := NewJWTManager(secret)
	if err != nil {
		t.Fatalf("NewJWTManager returned an error: %v", err)
	}

	user := models.User{
		ID:   primitive.NewObjectID(),
		Role: "admin",
	}
	tokenString, expiresAt, err := manager.Issue(user)
	if err != nil {
		t.Fatalf("Issue returned an error: %v", err)
	}

	claims := new(Claims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			t.Errorf("unexpected signing method %v", token.Method)
		}
		return secret, nil
	})
	if err != nil {
		t.Fatalf("parse signed token: %v", err)
	}
	if !token.Valid {
		t.Fatal("issued token is invalid")
	}
	if claims.Subject != user.ID.Hex() {
		t.Errorf("token subject = %q, want %q", claims.Subject, user.ID.Hex())
	}
	if claims.Role != user.Role {
		t.Errorf("token role = %q, want %q", claims.Role, user.Role)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(expiresAt) {
		t.Errorf("token expiration = %v, want %v", claims.ExpiresAt, expiresAt)
	}
	if time.Until(expiresAt) <= 0 || time.Until(expiresAt) > TokenLifetime {
		t.Errorf("token expiration %v is not within the expected lifetime", expiresAt)
	}
}

func TestNewJWTManagerRejectsShortSecret(t *testing.T) {
	if _, err := NewJWTManager([]byte("too-short")); err == nil {
		t.Fatal("NewJWTManager accepted a secret shorter than 32 bytes")
	}
}

func TestJWTManagerRejectsUserWithoutID(t *testing.T) {
	manager, err := NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("NewJWTManager returned an error: %v", err)
	}

	if _, _, err := manager.Issue(models.User{}); !errors.Is(err, ErrUserIDRequired) {
		t.Errorf("Issue error = %v, want %v", err, ErrUserIDRequired)
	}
}
