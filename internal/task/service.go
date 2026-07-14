package task

func NewTaskService(r *TaskRepository) *TaskService {
	return &TaskService{repo: r}
}

func (s *TaskService) GetAllTasks() []Task {
	return s.repo.GetAll()
}

func (s *TaskService) GetById(id int) (Task, error) {
	task, err := s.repo.GetById(id)
	if err != nil {
		return Task{}, err
	}

	return task, nil
}

func (s *TaskService) CreateTask(input CreateTaskRequest) Task {
	newTask := s.repo.Create(input)

	s.repo.tasks = append(s.repo.tasks, newTask)
	s.repo.nextID++

	return newTask
}
