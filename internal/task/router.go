package task

import "github.com/gin-gonic/gin"

func RegisterRouters(rg *gin.RouterGroup, repo *TaskRepository) {
	service := NewTaskService(repo)
	handler := &TaskHandler{service: service}

	group := rg.Group("/tasks")
	{
		group.GET("", handler.GetTasks)
		group.GET("/:id", handler.GetTask)
		group.POST("", handler.CreateTask)
	}
}
