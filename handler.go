package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var repository = NewTaskRepository()

func GetTasks(c *gin.Context) {
	list := repository.GetAll()

	c.JSON(http.StatusOK, ApiResponse{
		Status: "aoeu",
		Data:   list,
	})
}
