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

func mustObjectIDFromHex(t *testing.T, value string) primitive.ObjectID {
	t.Helper()
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		t.Fatalf("parse test ObjectID: %v", err)
	}
	return id
}
