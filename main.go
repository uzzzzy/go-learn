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

	repo := task.NewTaskRepository()

	task.RegisterRouters(&router.RouterGroup, repo)

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
