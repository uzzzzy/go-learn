package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"api/internal/common"
	"api/internal/task"
)

func setupRouter() *gin.Engine {
	router := gin.Default()

	repo := task.NewTaskRepository()

	router.GET("/health", health)

	router.GET("/tasks", repo.GetTasks)
	router.GET("/tasks/:id", repo.GetTask)
	router.POST("/tasks", repo.CreateTasks)

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
