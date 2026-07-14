package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApiStatus string

const (
	StatusSuccess ApiStatus = "success"
	StatusFailed  ApiStatus = "failed"
)

type ApiResponse struct {
	Status ApiStatus `json:"status"`
	Data   any       `json:"data,omitempty"`
}

func main() {
	router := gin.Default()

	router.GET("/health", health)

	router.GET("/tasks", GetTasks)

	router.Run()
}

func health(c *gin.Context) {
	response := ApiResponse{
		Status: "ok",
	}

	c.JSON(http.StatusOK, response)
}
