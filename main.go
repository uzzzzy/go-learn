package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"api/internal/response"
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

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,

		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Println("Server running on port :8080...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Serve error: %s\n", err)
	}
}

func health(c *gin.Context) {
	resp := response.ApiResponse[any]{
		Status: "ok",
	}

	c.JSON(http.StatusOK, resp)
}
