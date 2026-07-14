package task

import (
	"net/http"
	"strconv"

	"api/internal/common"

	"github.com/gin-gonic/gin"
)

func (h *TaskHandler) GetTasks(c *gin.Context) {
	tasks := h.service.GetAllTasks()

	c.JSON(http.StatusOK, common.ApiResponse[[]Task]{
		Status: common.StatusSuccess,
		Data:   tasks,
	})
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var input CreateTaskRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, common.ApiResponse[any]{
			Status: common.StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	task := h.service.CreateTask(input)

	c.JSON(http.StatusCreated, common.ApiResponse[Task]{
		Status: common.StatusSuccess,
		Data:   task,
	})
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.ApiResponse[any]{
			Status: common.StatusFailed,
			Error:  "Invalid ID",
		})
		return
	}

	task, err := h.service.GetById(id)

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
