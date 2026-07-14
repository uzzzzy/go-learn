package task

import (
	"net/http"
	"strconv"

	"api/internal/common"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	repo *TaskRepository
}

func (repo *TaskRepository) GetTasks(c *gin.Context) {
	list := repo.GetAll()

	c.JSON(http.StatusOK, common.ApiResponse[[]Task]{
		Status: common.StatusSuccess,
		Data:   list,
	})
}

func (repo *TaskRepository) CreateTasks(c *gin.Context) {
	var input CreateTaskRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.ApiResponse[any]{
			Status: common.StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	task := repo.Create(input)

	c.JSON(http.StatusCreated, common.ApiResponse[Task]{
		Status: common.StatusSuccess,
		Data:   task,
	})
}

func (repo *TaskRepository) GetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ApiResponse[any]{
			Status: common.StatusFailed,
			Error:  "Invalid ID",
		})
		return
	}

	task, err := repo.GetById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, common.ApiResponse[any]{
			Status: common.StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, common.ApiResponse[Task]{
		Status: common.StatusSuccess,
		Data:   task,
	})
}
