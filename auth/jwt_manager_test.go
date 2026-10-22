package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var testJWTSecret = []byte("0123456789abcdef0123456789abcdef")

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

func TestJWTManagerRejectsUserWithoutRole(t *testing.T) {
	manager, err := NewJWTManager(testJWTSecret)
	if err != nil {
		t.Fatalf("NewJWTManager returned an error: %v", err)
	}

	user := models.User{ID: primitive.NewObjectID()}
	if _, _, err := manager.Issue(user); !errors.Is(err, ErrRoleRequired) {
		t.Errorf("Issue error = %v, want %v", err, ErrRoleRequired)
	}
}

func TestJWTManagerVerifyAndMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager, err := NewJWTManager(testJWTSecret)
	if err != nil {
		t.Fatalf("NewJWTManager returned an error: %v", err)
	}
	user := models.User{ID: primitive.NewObjectID(), Role: "admin"}
	validToken, _, err := manager.Issue(user)
	if err != nil {
		t.Fatalf("Issue returned an error: %v", err)
	}

	expiredToken := signTestClaims(t, testJWTSecret, Claims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.Hex(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	tokenWithoutExpiration := signTestClaims(t, testJWTSecret, Claims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: user.ID.Hex(),
		},
	})
	invalidSubjectToken := signTestClaims(t, testJWTSecret, Claims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "not-an-object-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	wrongSignatureToken := signTestClaims(t, []byte("abcdef0123456789abcdef0123456789"), Claims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.Hex(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})

	tests := []struct {
		name          string
		authorization string
		wantStatus    int
		wantIdentity  bool
	}{
		{name: "valid bearer token", authorization: "Bearer " + validToken, wantStatus: http.StatusOK, wantIdentity: true},
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "malformed authorization header", authorization: "Bearer", wantStatus: http.StatusUnauthorized},
		{name: "wrong authorization scheme", authorization: "Basic " + validToken, wantStatus: http.StatusUnauthorized},
		{name: "malformed token", authorization: "Bearer not-a-jwt", wantStatus: http.StatusUnauthorized},
		{name: "expired token", authorization: "Bearer " + expiredToken, wantStatus: http.StatusUnauthorized},
		{name: "missing expiration", authorization: "Bearer " + tokenWithoutExpiration, wantStatus: http.StatusUnauthorized},
		{name: "invalid signature", authorization: "Bearer " + wrongSignatureToken, wantStatus: http.StatusUnauthorized},
		{name: "invalid subject", authorization: "Bearer " + invalidSubjectToken, wantStatus: http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlerReached := false
			router := gin.New()
			router.GET("/protected", manager.Middleware(), func(c *gin.Context) {
				handlerReached = true
				if got := c.GetString(AuthenticatedUserIDKey); got != user.ID.Hex() {
					t.Errorf("authenticated user ID = %q, want %q", got, user.ID.Hex())
				}
				if got := c.GetString(AuthenticatedRoleKey); got != user.Role {
					t.Errorf("authenticated role = %q, want %q", got, user.Role)
				}
				c.Status(http.StatusOK)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if handlerReached != test.wantIdentity {
				t.Errorf("protected handler reached = %v, want %v", handlerReached, test.wantIdentity)
			}
			if !test.wantIdentity && !strings.Contains(response.Body.String(), "Unauthorized") {
				t.Errorf("unauthorized response should be generic: %s", response.Body.String())
			}
		})
	}
}

func signTestClaims(t *testing.T, secret []byte, claims Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return signed
}
