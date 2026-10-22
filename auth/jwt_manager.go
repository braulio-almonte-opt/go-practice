package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/golang-jwt/jwt/v5"
)

const TokenLifetime = time.Hour

var ErrUserIDRequired = errors.New("user ID is required to issue a token")

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret []byte
}

func NewJWTManager(secret []byte) (*JWTManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT signing secret must contain at least 32 bytes")
	}
	return &JWTManager{secret: append([]byte(nil), secret...)}, nil
}

func (m *JWTManager) Issue(user models.User) (string, time.Time, error) {
	if user.ID.IsZero() {
		return "", time.Time{}, ErrUserIDRequired
	}

	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(TokenLifetime)
	claims := Claims{
		Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.Hex(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign JWT: %w", err)
	}
	return signedToken, expiresAt, nil
}
