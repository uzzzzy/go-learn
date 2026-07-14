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

type ApiResponse[T any] struct {
	Status ApiStatus `json:"status"`
	Data   T         `json:"data,omitempty"`
	Error  string    `json:"error,omitempty"`
}

func setupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/health", health)

	router.GET("/tasks", GetTasks)
	router.GET("/tasks/:id", GetTask)
	router.POST("/tasks", CreateTasks)

	return router
}

func main() {
	router := setupRouter()

	router.Run()
}

func health(c *gin.Context) {
	response := ApiResponse[any]{
		Status: "ok",
	}

	c.JSON(http.StatusOK, response)
}
