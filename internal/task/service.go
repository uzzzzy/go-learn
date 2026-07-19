package task

type TaskService struct {
	repo Repository
}

func NewTaskService(r Repository) *TaskService {
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
	return s.repo.Create(input)
}

func (s *TaskService) UpdateTaskById(id int, input UpdateTaskRequest) (Task, error) {
	return s.repo.UpdateById(id, input)
}

func (s *TaskService) DeleteById(id int) (Task, error) {
	return s.repo.DeleteById(id)
}
