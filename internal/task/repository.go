package task

import (
	"errors"
	"fmt"
	"sync"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskRepository struct {
	mu     sync.RWMutex
	tasks  []Task
	nextID int
}

func NewRepository() *TaskRepository {
	return &TaskRepository{
		tasks:  []Task{},
		nextID: 1,
	}
}

func (r *TaskRepository) Create(payload CreateTaskRequest) Task {
	r.mu.Lock()
	defer r.mu.Unlock()

	newTask := Task{
		Id:        r.nextID,
		Title:     payload.Title,
		Completed: false,
	}

	fmt.Println(newTask.LogWithAction("CREATE"))

	r.tasks = append(r.tasks, newTask)
	r.nextID++

	return newTask
}

func (r *TaskRepository) GetAll() []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Task, len(r.tasks))
	copy(out, r.tasks)

	return out
}

func (r *TaskRepository) GetById(id int) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, task := range r.tasks {
		if task.Id == id {
			fmt.Println(task.LogWithAction("GET"))
			return task, nil
		}
	}

	return Task{}, ErrTaskNotFound
}

func (r *TaskRepository) UpdateById(id int, payload UpdateTaskRequest) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, task := range r.tasks {
		if task.Id == id {
			r.tasks[i].Title = payload.Title
			r.tasks[i].Completed = payload.Completed

			fmt.Println(r.tasks[i].LogWithAction("UPDATE"))

			return r.tasks[i], nil
		}
	}

	return Task{}, ErrTaskNotFound
}

func (r *TaskRepository) DeleteById(id int) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, task := range r.tasks {
		if task.Id == id {
			deletedTask := task
			r.tasks = append(r.tasks[:i], r.tasks[i+1:]...)

			fmt.Println(task.LogWithAction("DELETE"))

			return deletedTask, nil
		}
	}

	return Task{}, ErrTaskNotFound
}
