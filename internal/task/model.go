package task

type TaskHandler struct {
	service *TaskService
}

type TaskService struct {
	repo *TaskRepository
}

type Task struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type CreateTaskRequest struct {
	Title string
}

type UpdateTaskRequest struct {
	Title     string
	Completed bool
}
