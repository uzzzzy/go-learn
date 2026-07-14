package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"api/internal/common"
	"api/internal/task"
)

func setupRouter() *gin.Engine {
	router := gin.Default()

	apiGroup := router.Group("/api/v1")

	taskRepo := task.NewTaskRepository()
	task.RegisterRouters(apiGroup, taskRepo)

	router.GET("/health", health)

	return router
}

func main() {
	router := setupRouter()

	router.Run()
}

func health(c *gin.Context) {
	response := common.ApiResponse[any]{
		Status: "ok",
	}

	c.JSON(http.StatusOK, response)
}
