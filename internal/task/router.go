package task

import "github.com/gin-gonic/gin"

func RegisterRouters(rg *gin.RouterGroup, repo Repository) {
	service := NewTaskService(repo)
	handler := NewTaskHandler(service)

	group := rg.Group("/tasks")
	{
		group.GET("", handler.GetTasks)
		group.GET("/:id", handler.GetTask)
		group.POST("", handler.CreateTask)
	}
}
