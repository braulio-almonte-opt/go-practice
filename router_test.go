package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/braulio-almonte-opt/go-practice/auth"
	"github.com/braulio-almonte-opt/go-practice/models"
	repositorymocks "github.com/braulio-almonte-opt/go-practice/repository/mocks"
	"github.com/braulio-almonte-opt/go-practice/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRouterProtectsUserRoutesAndResolvesCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}

	userID := primitive.NewObjectID()
	user := models.User{ID: userID, UserName: "jane", Password: "secret", Role: "user"}
	repo := new(repositorymocks.MockUserRepository)
	repo.EXPECT().FindByID(mock.Anything, userID.Hex()).Return(user, nil).Once()
	router := newRouter(service.NewUserService(repo, jwtManager), jwtManager)

	for _, path := range []string{"/users/me", "/users/" + userID.Hex(), "/users"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without token status = %d, want %d", path, response.Code, http.StatusUnauthorized)
		}
	}

	token, _, err := jwtManager.Issue(user)
	if err != nil {
		t.Fatalf("issue test token: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /users/me status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if response.Body.String() == "" || strings.Contains(response.Body.String(), `"password"`) {
		t.Errorf("GET /users/me returned an empty response or exposed a password: %s", response.Body.String())
	}
	repo.AssertExpectations(t)
}
