package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApiResponse struct {
	Status string `json:"status"`
}

func main() {
	router := gin.Default()

	router.GET("/health", health)

	router.Run()
}

func health(c *gin.Context) {
	response := ApiResponse{
		Status: "ok",
	}

	c.JSON(http.StatusOK, response)
}
