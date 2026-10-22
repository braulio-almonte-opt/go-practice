package test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/braulio-almonte-opt/go-practice/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestPostUser(t *testing.T) {
	var insertedUser models.User
	router := setupRouter(func(user models.User) error {
		insertedUser = user
		return nil
	})

	requestBody := `{"username":"test_name","password":"test_password"}`
	request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
	}
	if insertedUser.UserName != "test_name" || insertedUser.Password != "test_password" {
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
	router := setupRouter(func(models.User) error {
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
	router := setupRouter(func(models.User) error {
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
	router := setupListUsersRouter(func() ([]models.User, error) {
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
	router := setupFindUserRouter(func(id string) (models.User, error) {
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
	router := setupListUsersRouter(func() ([]models.User, error) {
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
			router := setupFindUserRouter(func(string) (models.User, error) {
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
