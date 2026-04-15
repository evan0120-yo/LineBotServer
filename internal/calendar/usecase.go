package calendar

import "context"

// UseCase orchestrates calendar task operations.
type UseCase struct {
	service    *Service
	repository *Repository
}

// NewUseCase creates a new calendar UseCase.
func NewUseCase(service *Service, repository *Repository) *UseCase {
	return &UseCase{
		service:    service,
		repository: repository,
	}
}

// Create creates a new calendar task.
// Validates the command, persists to Firestore, and returns the created task.
func (u *UseCase) Create(ctx context.Context, command CreateCommand) (CalendarTask, error) {
	// Validate required fields
	if err := u.service.ValidateCreate(command); err != nil {
		return CalendarTask{}, err
	}

	// Persist to Firestore
	task, err := u.repository.Create(ctx, command)
	if err != nil {
		return CalendarTask{}, err
	}

	return task, nil
}
