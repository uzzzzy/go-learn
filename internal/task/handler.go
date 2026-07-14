package task

import (
	"errors"
	"net/http"
	"strconv"

	"api/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *TaskHandler) GetTasks(c *gin.Context) {
	tasks := h.service.GetAllTasks()

	c.JSON(http.StatusOK, response.ApiResponse[[]Task]{
		Status: response.StatusSuccess,
		Data:   tasks,
	})
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var input CreateTaskRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse[any]{
			Status: response.StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	task := h.service.CreateTask(input)

	c.JSON(http.StatusCreated, response.ApiResponse[Task]{
		Status: response.StatusSuccess,
		Data:   task,
	})
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.ApiResponse[any]{
			Status: response.StatusFailed,
			Error:  "Invalid ID",
		})
		return
	}

	task, err := h.service.GetById(id)

	if errors.Is(err, ErrTaskNotFound) {
		c.JSON(http.StatusNotFound, response.ApiResponse[any]{
			Status: response.StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, response.ApiResponse[any]{
			Status: response.StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response.ApiResponse[Task]{
		Status: response.StatusSuccess,
		Data:   task,
	})
}

func NewTaskHandler(s Service) *TaskHandler {
	return &TaskHandler{service: s}
}
