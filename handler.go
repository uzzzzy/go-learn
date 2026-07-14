package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var repository = NewTaskRepository()

func GetTasks(c *gin.Context) {
	list := repository.GetAll()

	c.JSON(http.StatusOK, ApiResponse[[]Task]{
		Status: StatusSuccess,
		Data:   list,
	})
}

func CreateTasks(c *gin.Context) {
	var input CreateTaskRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse[any]{
			Status: StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	task := repository.Create(input)

	c.JSON(http.StatusCreated, ApiResponse[Task]{
		Status: StatusSuccess,
		Data:   task,
	})
}

func GetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse[any]{
			Status: StatusFailed,
			Error:  "Invalid ID",
		})
		return
	}

	task, err := repository.GetById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ApiResponse[any]{
			Status: StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse[Task]{
		Status: StatusSuccess,
		Data:   task,
	})
}
