package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/braulio-almonte-opt/go-practice/auth"
	"github.com/braulio-almonte-opt/go-practice/models"
	repositorymocks "github.com/braulio-almonte-opt/go-practice/repository/mocks"
	"github.com/braulio-almonte-opt/go-practice/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestRouterAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}

	ownerID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()
	tests := []struct {
		name       string
		method     string
		path       string
		subjectID  primitive.ObjectID
		role       string
		wantStatus int
		wantFindID string
		wantList   bool
	}{
		{
			name:       "user may fetch own record",
			method:     http.MethodGet,
			path:       "/users/" + ownerID.Hex(),
			subjectID:  ownerID,
			role:       models.RoleUser,
			wantStatus: http.StatusOK,
			wantFindID: ownerID.Hex(),
		},
		{
			name:       "user may not fetch another user's record",
			method:     http.MethodGet,
			path:       "/users/" + otherID.Hex(),
			subjectID:  ownerID,
			role:       models.RoleUser,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin may fetch another user's record",
			method:     http.MethodGet,
			path:       "/users/" + otherID.Hex(),
			subjectID:  ownerID,
			role:       models.RoleAdmin,
			wantStatus: http.StatusOK,
			wantFindID: otherID.Hex(),
		},
		{
			name:       "regular user may not list users",
			method:     http.MethodGet,
			path:       "/users",
			subjectID:  ownerID,
			role:       models.RoleUser,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "admin may list users",
			method:     http.MethodGet,
			path:       "/users",
			subjectID:  ownerID,
			role:       models.RoleAdmin,
			wantStatus: http.StatusOK,
			wantList:   true,
		},
		{
			name:       "current user route stays accessible to regular users",
			method:     http.MethodGet,
			path:       "/users/me",
			subjectID:  ownerID,
			role:       models.RoleUser,
			wantStatus: http.StatusOK,
			wantFindID: ownerID.Hex(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := new(repositorymocks.MockUserRepository)
			if test.wantFindID != "" {
				user := models.User{
					ID:       mustObjectIDFromHex(t, test.wantFindID),
					UserName: "jane",
					Password: "must-not-be-returned",
					Role:     models.RoleUser,
				}
				repo.EXPECT().
					FindByID(mock.Anything, test.wantFindID).
					Return(user, nil).
					Once()
			}
			if test.wantList {
				repo.EXPECT().
					FindAll(mock.Anything).
					Return([]models.User{{ID: ownerID, UserName: "jane", Password: "secret"}}, nil).
					Once()
			}

			userService := service.NewUserService(repo, jwtManager)
			router := newRouter(userService, jwtManager)
			token, _, err := jwtManager.Issue(models.User{ID: test.subjectID, Role: test.role})
			if err != nil {
				t.Fatalf("issue test token: %v", err)
			}
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus == http.StatusForbidden &&
				!strings.Contains(response.Body.String(), "Forbidden") {
				t.Errorf("forbidden response body = %q", response.Body.String())
			}
			if strings.Contains(response.Body.String(), `"password"`) ||
				strings.Contains(response.Body.String(), "must-not-be-returned") ||
				strings.Contains(response.Body.String(), "secret") {
				t.Errorf("response leaked password data: %s", response.Body.String())
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestRouterRejectsInvalidTokensOnProtectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}
	user := models.User{ID: primitive.NewObjectID(), Role: models.RoleAdmin}
	expiredToken, err := signRouterTestToken(
		[]byte("0123456789abcdef0123456789abcdef"),
		auth.Claims{
			Role: models.RoleAdmin,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   user.ID.Hex(),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			},
		},
	)
	if err != nil {
		t.Fatalf("create expired test token: %v", err)
	}

	repo := new(repositorymocks.MockUserRepository)
	router := newRouter(service.NewUserService(repo, jwtManager), jwtManager)
	tests := []struct {
		name          string
		path          string
		authorization string
	}{
		{name: "missing token", path: "/users/me"},
		{name: "malformed token", path: "/users", authorization: "Bearer not-a-jwt"},
		{name: "expired token", path: "/users/" + user.ID.Hex(), authorization: "Bearer " + expiredToken},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Errorf("GET %s status = %d, want %d: %s", test.path, response.Code, http.StatusUnauthorized, response.Body.String())
			}
		})
	}
	repo.AssertExpectations(t)
}

func TestRouterProtectedEndpointsMapRepositoryErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}
	userID := primitive.NewObjectID()
	tests := []struct {
		name       string
		path       string
		role       string
		findID     string
		findErr    error
		listErr    error
		wantStatus int
	}{
		{
			name:       "invalid ID",
			path:       "/users/not-an-object-id",
			role:       models.RoleAdmin,
			findID:     "not-an-object-id",
			findErr:    models.ErrInvalidUserID,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing user",
			path:       "/users/" + userID.Hex(),
			role:       models.RoleUser,
			findID:     userID.Hex(),
			findErr:    mongo.ErrNoDocuments,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "user lookup failure",
			path:       "/users/" + userID.Hex(),
			role:       models.RoleUser,
			findID:     userID.Hex(),
			findErr:    errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "list failure",
			path:       "/users",
			role:       models.RoleAdmin,
			listErr:    errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := new(repositorymocks.MockUserRepository)
			if test.findID != "" {
				repo.EXPECT().
					FindByID(mock.Anything, test.findID).
					Return(models.User{}, test.findErr).
					Once()
			}
			if test.listErr != nil {
				repo.EXPECT().
					FindAll(mock.Anything).
					Return(nil, test.listErr).
					Once()
			}
			router := newRouter(service.NewUserService(repo, jwtManager), jwtManager)
			token, _, err := jwtManager.Issue(models.User{ID: userID, Role: test.role})
			if err != nil {
				t.Fatalf("issue test token: %v", err)
			}
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Errorf("GET %s status = %d, want %d: %s", test.path, response.Code, test.wantStatus, response.Body.String())
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestRouterRegistrationAndLoginRemainPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}
	user := models.User{
		ID:       primitive.NewObjectID(),
		UserName: "jane",
		Password: "password",
		Role:     models.RoleUser,
	}
	repo := new(repositorymocks.MockUserRepository)
	repo.EXPECT().
		InsertUser(mock.Anything, models.User{UserName: user.UserName, Password: user.Password, Role: models.RoleUser}).
		Return(nil).
		Once()
	repo.EXPECT().FindByUsername(mock.Anything, user.UserName).Return(user, nil).Once()
	router := newRouter(service.NewUserService(repo, jwtManager), jwtManager)

	registerRequest := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"username":"jane","password":"password","role":"admin"}`),
	)
	registerRequest.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	router.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Errorf("public registration status = %d, want %d: %s", registerResponse.Code, http.StatusCreated, registerResponse.Body.String())
	}

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"jane","password":"password"}`),
	)
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	router.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Errorf("public login status = %d, want %d: %s", loginResponse.Code, http.StatusOK, loginResponse.Body.String())
	}
	repo.AssertExpectations(t)
}

func TestRouterLogoutRequiresJWTAndDoesNotRevokeToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}
	repo := new(repositorymocks.MockUserRepository)
	router := newRouter(service.NewUserService(repo, jwtManager), jwtManager)
	user := models.User{ID: primitive.NewObjectID(), Role: models.RoleUser}
	token, _, err := jwtManager.Issue(user)
	if err != nil {
		t.Fatalf("issue test token: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/logout", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("logout without token status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	request = httptest.NewRequest(http.MethodPost, "/logout", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if strings.Contains(response.Body.String(), token) {
		t.Errorf("logout response must not echo the token: %s", response.Body.String())
	}

	if _, err := jwtManager.Verify(token); err != nil {
		t.Errorf("stateless logout should not revoke the token before expiration: %v", err)
	}
}

func signRouterTestToken(secret []byte, claims auth.Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func mustObjectIDFromHex(t *testing.T, value string) primitive.ObjectID {
	t.Helper()
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		t.Fatalf("parse test ObjectID: %v", err)
	}
	return id
}
