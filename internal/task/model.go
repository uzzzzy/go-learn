package task

type Repository interface {
	Create(payload CreateTaskRequest) Task
	GetAll() []Task
	GetById(id int) (Task, error)
	UpdateById(id int, payload UpdateTaskRequest) (Task, error)
	DeleteById(id int) (Task, error)
}

type Service interface {
	GetAllTasks() []Task
	GetById(id int) (Task, error)
	CreateTask(input CreateTaskRequest) Task
}

type TaskService struct {
	repo Repository
}

type Task struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type CreateTaskRequest struct {
	Title string `json:"title" binding:"required,max=256"`
}

type UpdateTaskRequest struct {
	Title     string `json:"title" binding:"required"`
	Completed bool   `json:"completed"`
}
