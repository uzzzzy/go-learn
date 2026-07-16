package task

type CreateTaskRequest struct {
	Title string `json:"title" binding:"required,max=256"`
}

type UpdateTaskRequest struct {
	Title     string `json:"title" binding:"required,max=256"`
	Completed bool   `json:"completed"`
}
