package startups

import "context"

type StartupService interface {
	CreateStartup(ctx context.Context, input Startup) (Startup, error)
	UpdateStartup(ctx context.Context, input Startup) (Startup, error)
	DeleteStartup(ctx context.Context, id int64) error
	DeleteAllStartups(ctx context.Context) error
	GetStartupByID(ctx context.Context, id int64) (Startup, error)
	ListStartups(ctx context.Context, page, limit int) ([]Startup, int64, error)
	ListStartupsByUser(ctx context.Context, uuid string) ([]Startup, error)
}

type startupService struct {
	repo StartupRepository
}

func NewStartupService(repo StartupRepository) StartupService {
	return &startupService{repo: repo}
}

func (s *startupService) CreateStartup(ctx context.Context, input Startup) (Startup, error) {
	if input.Status == "" {
		input.Status = "failed"
	}
	return s.repo.CreateStartup(ctx, input)
}

func (s *startupService) UpdateStartup(ctx context.Context, input Startup) (Startup, error) {
	if input.Status == "" {
		input.Status = "failed"
	}
	return s.repo.UpdateStartup(ctx, input)
}

func (s *startupService) DeleteStartup(ctx context.Context, id int64) error {
	return s.repo.DeleteStartup(ctx, id)
}

func (s *startupService) GetStartupByID(ctx context.Context, id int64) (Startup, error) {
	return s.repo.GetStartupByID(ctx, id)
}

func (s *startupService) ListStartups(ctx context.Context, page, limit int) ([]Startup, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.ListStartups(ctx, limit, offset)
}

func (s *startupService) DeleteAllStartups(ctx context.Context) error {
	return s.repo.DeleteAllStartups(ctx)
}

func (s *startupService) ListStartupsByUser(ctx context.Context, uuid string) ([]Startup, error) {
	return s.repo.ListStartupsByUser(ctx, uuid)
}
