package task

import "fmt"

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

type Task struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func (t Task) String() string {
	return fmt.Sprintf("Task ID: %d, Title: %q, Completed: %t", t.Id, t.Title, t.Completed)
}

func (t Task) LogWithAction(action string) string {
	return fmt.Sprintf("[%s] ID: %d, Title: %q, Completed: %t", action, t.Id, t.Title, t.Completed)
}
