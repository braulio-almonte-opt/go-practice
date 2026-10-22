package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/braulio-almonte-opt/go-practice/auth"
	"github.com/braulio-almonte-opt/go-practice/models"
	"github.com/braulio-almonte-opt/go-practice/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestProtectedUserRoutesRequireJWTAndMeUsesTokenIdentity(t *testing.T) {
	jwtManager, err := auth.NewJWTManager([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}
	userID := primitive.NewObjectID()
	token, _, err := jwtManager.Issue(models.User{ID: userID, Role: "user"})
	if err != nil {
		t.Fatalf("issue test token: %v", err)
	}

	findCalled := false
	listCalled := false
	router := setupProtectedUsersRouter(
		jwtManager,
		func(_ context.Context, requestedID string) (models.User, error) {
			findCalled = true
			if requestedID != userID.Hex() {
				t.Errorf("lookup ID = %q, want verified token subject %q", requestedID, userID.Hex())
			}
			return models.User{
				ID:       userID,
				UserName: "jane",
				Password: "must-not-be-returned",
				Role:     "user",
			}, nil
		},
		func(context.Context) ([]models.User, error) {
			listCalled = true
			return []models.User{}, nil
		},
	)

	for _, path := range []string{"/users/me", "/users/" + userID.Hex(), "/users"} {
		t.Run("rejects missing token "+path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d for %s", response.Code, http.StatusUnauthorized, path)
			}
		})
	}
	if findCalled || listCalled {
		t.Fatal("protected handlers ran without a valid token")
	}

	request := httptest.NewRequest(http.MethodGet, "/users/me?id="+primitive.NewObjectID().Hex(), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /users/me status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !findCalled {
		t.Fatal("GET /users/me did not look up the user from the token")
	}
	if strings.Contains(response.Body.String(), "must-not-be-returned") ||
		strings.Contains(response.Body.String(), `"password"`) {
		t.Errorf("GET /users/me leaked password data: %s", response.Body.String())
	}

	findCalled = false
	for _, test := range []struct {
		path       string
		wantStatus int
	}{
		{path: "/users/" + userID.Hex(), wantStatus: http.StatusOK},
		{path: "/users", wantStatus: http.StatusForbidden},
	} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != test.wantStatus {
			t.Errorf("GET %s status = %d, want %d: %s", test.path, response.Code, test.wantStatus, response.Body.String())
		}
	}
	if !findCalled || listCalled {
		t.Error("owner lookup should run; user-list lookup must be blocked for a regular user")
	}
}

func TestPostLoginReturnsBearerToken(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	router := setupLoginRouter(func(_ context.Context, username, password string) (service.LoginResult, error) {
		if username != "jane" || password != "correct-password" {
			t.Fatalf("login received credentials (%q, %q)", username, password)
		}
		return service.LoginResult{AccessToken: "signed-token", ExpiresAt: expiresAt}, nil
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"username":"jane","password":"correct-password"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
	if body["access_token"] != "signed-token" || body["token_type"] != "Bearer" {
		t.Errorf("unexpected login response: %#v", body)
	}
	if body["expires_at"] != expiresAt.Format(time.RFC3339) {
		t.Errorf("expires_at = %#v, want %q", body["expires_at"], expiresAt.Format(time.RFC3339))
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Errorf("login response must not contain password data: %s", response.Body.String())
	}
}

func TestPostLoginMapsInputAndCredentialErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		loginErr   error
		wantStatus int
		wantError  string
	}{
		{
			name:       "malformed JSON",
			body:       `{"username":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Username and password are required",
		},
		{
			name:       "missing password",
			body:       `{"username":"jane"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "Username and password are required",
		},
		{
			name:       "invalid credentials",
			body:       `{"username":"jane","password":"wrong-password"}`,
			loginErr:   service.ErrInvalidCredentials,
			wantStatus: http.StatusUnauthorized,
			wantError:  "Invalid username or password",
		},
		{
			name:       "authentication backend failure",
			body:       `{"username":"jane","password":"password"}`,
			loginErr:   errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to authenticate user",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loginCalled := false
			router := setupLoginRouter(func(context.Context, string, string) (service.LoginResult, error) {
				loginCalled = true
				return service.LoginResult{}, test.loginErr
			})
			request := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", test.wantStatus, response.Code, response.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response JSON: %v", err)
			}
			if body["error"] != test.wantError {
				t.Errorf("error = %q, want %q", body["error"], test.wantError)
			}
			if test.wantStatus == http.StatusBadRequest && loginCalled {
				t.Error("login callback must not be called for invalid request input")
			}
		})
	}
}

func TestPostUser(t *testing.T) {
	var insertedUser models.User
	router := setupRouter(func(_ context.Context, user models.User) error {
		insertedUser = user
		return nil
	})

	requestBody := `{"username":"test_name","password":"test_password","role":"admin"}`
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	if insertedUser.UserName != "test_name" ||
		insertedUser.Password != "test_password" ||
		insertedUser.Role != models.RoleUser {
		t.Fatalf("inserted unexpected user: %+v", insertedUser)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
	if body["message"] != "User inserted" {
		t.Errorf("expected success message, got %q", body["message"])
	}
}

func TestPostUserRejectsInvalidJSON(t *testing.T) {
	insertCalled := false
	router := setupRouter(func(context.Context, models.User) error {
		insertCalled = true
		return nil
	})

	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"username":`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, response.Code, response.Body.String())
	}
	if insertCalled {
		t.Error("insert function should not be called for invalid JSON")
	}
}

func TestPostUserReturnsInternalServerErrorWhenInsertFails(t *testing.T) {
	router := setupRouter(func(context.Context, models.User) error {
		return errors.New("database unavailable")
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"username":"test_name","password":"test_password"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d: %s", http.StatusInternalServerError, response.Code, response.Body.String())
	}
}

func TestListUsersDoesNotExposePasswords(t *testing.T) {
	router := setupListUsersRouter(func(context.Context) ([]models.User, error) {
		return []models.User{
			{
				ID:       primitive.NewObjectID(),
				UserName: "test_name",
				Password: "secret-password",
				RealName: "Test",
				LastName: "User",
				Balance:  12.5,
				Role:     "member",
			},
		}, nil
	})

	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "password") || strings.Contains(response.Body.String(), "secret-password") {
		t.Fatalf("list response leaked password data: %s", response.Body.String())
	}

	var users []map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &users); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
	if len(users) != 1 || users[0]["username"] != "test_name" {
		t.Fatalf("unexpected list response: %#v", users)
	}
}

func TestFindUserDoesNotExposePassword(t *testing.T) {
	userID := primitive.NewObjectID()
	router := setupFindUserRouter(func(_ context.Context, id string) (models.User, error) {
		if id != userID.Hex() {
			t.Fatalf("expected lookup ID %q, got %q", userID.Hex(), id)
		}
		return models.User{
			ID:       userID,
			UserName: "test_name",
			Password: "secret-password",
			RealName: "Test",
			LastName: "User",
			Balance:  12.5,
			Role:     "member",
		}, nil
	})

	request := httptest.NewRequest(http.MethodGet, "/users/"+userID.Hex(), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "password") || strings.Contains(response.Body.String(), "secret-password") {
		t.Fatalf("detail response leaked password data: %s", response.Body.String())
	}

	var user map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &user); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
	if user["username"] != "test_name" {
		t.Fatalf("unexpected detail response: %#v", user)
	}
}

func TestListUsersReturnsInternalServerErrorWhenFindFails(t *testing.T) {
	router := setupListUsersRouter(func(context.Context) ([]models.User, error) {
		return nil, errors.New("database unavailable")
	})

	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d: %s", http.StatusInternalServerError, response.Code, response.Body.String())
	}
}

func TestFindUserMapsErrorsToHTTPStatuses(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantError  string
	}{
		{
			name:       "invalid ID",
			err:        models.ErrInvalidUserID,
			wantStatus: http.StatusBadRequest,
			wantError:  "Invalid user ID",
		},
		{
			name:       "user not found",
			err:        mongo.ErrNoDocuments,
			wantStatus: http.StatusNotFound,
			wantError:  "User not found",
		},
		{
			name:       "database failure",
			err:        errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantError:  "Failed to find user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupFindUserRouter(func(context.Context, string) (models.User, error) {
				return models.User{}, tt.err
			})

			request := httptest.NewRequest(http.MethodGet, "/users/test-id", nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, response.Code, response.Body.String())
			}

			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response JSON: %v", err)
			}
			if body["error"] != tt.wantError {
				t.Errorf("expected error %q, got %q", tt.wantError, body["error"])
			}
		})
	}
}
