package test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/braulio-almonte-opt/go-practice/models"
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
