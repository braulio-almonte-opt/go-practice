package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const TokenLifetime = time.Hour

const (
	AuthenticatedUserIDKey = "authenticatedUserID"
	AuthenticatedRoleKey   = "authenticatedRole"
)

var (
	ErrUserIDRequired = errors.New("user ID is required to issue a token")
	ErrRoleRequired   = errors.New("user role is required to issue a token")
	ErrInvalidToken   = errors.New("invalid JWT")
)

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
	if user.Role == "" {
		return "", time.Time{}, ErrRoleRequired
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

func (m *JWTManager) Verify(tokenString string) (*Claims, error) {
	claims := new(Claims)
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !token.Valid || claims.Subject == "" || claims.Role == "" {
		return nil, ErrInvalidToken
	}
	if _, err := primitive.ObjectIDFromHex(claims.Subject); err != nil {
		return nil, fmt.Errorf("%w: invalid subject", ErrInvalidToken)
	}
	return claims, nil
}

func (m *JWTManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		const unauthorizedMessage = "Unauthorized"

		header := strings.Fields(c.GetHeader("Authorization"))
		if len(header) != 2 || !strings.EqualFold(header[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": unauthorizedMessage})
			return
		}

		claims, err := m.Verify(header[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": unauthorizedMessage})
			return
		}

		c.Set(AuthenticatedUserIDKey, claims.Subject)
		c.Set(AuthenticatedRoleKey, claims.Role)
		c.Next()
	}
}
