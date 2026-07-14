package task

import "github.com/gin-gonic/gin"

func RegisterRouters(rg *gin.RouterGroup, repo *TaskRepository) {
	group := rg.Group("/tasks")
	{
		group.GET("", repo.GetTask)
		group.GET("/:id", repo.GetTask)
		group.POST("", repo.CreateTasks)
	}
}
