package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var repository = NewTaskRepository()

func GetTasks(c *gin.Context) {
	list := repository.GetAll()

	c.JSON(http.StatusOK, ApiResponse{
		Status: StatusSuccess,
		Data:   list,
	})
}

func CreateTasks(c *gin.Context) {
	var input CreateTaskRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Status: StatusFailed,
			Error:  err.Error(),
		})
	}

	repository.Create(input)

	c.JSON(http.StatusCreated, ApiResponse{
		Status: StatusSuccess,
		Data:   input,
	})
}
