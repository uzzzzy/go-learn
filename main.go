package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"api/internal/common"
	"api/internal/task"
)

func setupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/health", health)

	router.GET("/tasks", task.GetTasks)
	router.GET("/tasks/:id", task.GetTask)
	router.POST("/tasks", task.CreateTasks)

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
