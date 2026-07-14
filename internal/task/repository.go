package task

import (
	"errors"
)

type TaskRepository struct {
	tasks  []Task
	nextID int
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks:  []Task{},
		nextID: 1,
	}
}

func (r *TaskRepository) Create(payload CreateTaskRequest) Task {
	newTask := Task{
		Id:        r.nextID,
		Title:     payload.Title,
		Completed: false,
	}

	r.tasks = append(r.tasks, newTask)
	r.nextID++
	return newTask
}

func (r *TaskRepository) GetAll() []Task {
	return r.tasks
}

func (r *TaskRepository) GetById(id int) (Task, error) {
	for _, task := range r.tasks {
		if task.Id == id {
			return task, nil
		}
	}

	return Task{}, errors.New("Task Not Found")
}

func (r *TaskRepository) UpdateById(id int, payload UpdateTaskRequest) (Task, error) {
	for i, task := range r.tasks {
		if task.Id == id {
			r.tasks[i].Title = payload.Title
			r.tasks[i].Completed = payload.Completed

			return r.tasks[i], nil
		}
	}

	return Task{}, errors.New("Task Not Found")
}

func (r *TaskRepository) DeleteById(id int) (Task, error) {
	for i, task := range r.tasks {
		if task.Id == id {
			deletedTask := task
			r.tasks = append(r.tasks[:i], r.tasks[i+1:]...)
			return deletedTask, nil
		}
	}

	return Task{}, errors.New("Task Not Found")
}
