package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"api/internal/response"
	"api/internal/task"
)

func TestTaskWorkflow(t *testing.T) {
	router := setupRouter()

	var createdTaskID int

	t.Run("Create Task", func(t *testing.T) {
		w := httptest.NewRecorder()
		jsonBody := []byte(`{"title": "Belajar Integration Testing"}`)

		req, _ := http.NewRequest("POST", "/api/v1/tasks", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp response.ApiResponse[task.Task]

		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Nil(t, err)
		assert.NotEmpty(t, resp.Data.Id)

		createdTaskID = resp.Data.Id
	})

	t.Run("Get All Task", func(t *testing.T) {
		r := httptest.NewRecorder()

		req, _ := http.NewRequest("GET", "/api/v1/tasks", nil)
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(r, req)

		assert.Equal(t, http.StatusOK, r.Code)

		var resp response.ApiResponse[[]task.Task]

		err := json.Unmarshal(r.Body.Bytes(), &resp)
		assert.Nil(t, err)
		assert.NotEmpty(t, resp.Data)

		assert.Equal(t, 1, len(*resp.Data))
	})

	t.Run("Get By Id", func(t *testing.T) {
		w := httptest.NewRecorder()

		url := fmt.Sprintf("/api/v1/tasks/%d", createdTaskID)

		req, _ := http.NewRequest("GET", url, nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp response.ApiResponse[task.Task]

		err := json.Unmarshal(w.Body.Bytes(), &resp)

		assert.Nil(t, err)
		assert.Equal(t, createdTaskID, resp.Data.Id)
	})

	_ = createdTaskID
}
