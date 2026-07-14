package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApiResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	router := gin.Default()

	router.GET("/health", health)

	router.Run()
}

func health(c *gin.Context) {
	response := ApiResponse{
		Status:  "success",
		Message: "Api Healthy",
	}

	c.JSON(http.StatusOK, response)
}
