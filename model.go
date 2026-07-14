package main

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
