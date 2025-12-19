package startups

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockStartupRepository struct {
	mock.Mock
}

func (m *mockStartupRepository) CreateStartup(ctx context.Context, input Startup) (Startup, error) {
	args := m.Called(ctx, input)
	startup, _ := args.Get(0).(Startup)
	return startup, args.Error(1)
}

func (m *mockStartupRepository) UpdateStartup(ctx context.Context, input Startup) (Startup, error) {
	args := m.Called(ctx, input)
	startup, _ := args.Get(0).(Startup)
	return startup, args.Error(1)
}

func (m *mockStartupRepository) DeleteStartup(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockStartupRepository) GetStartupByID(ctx context.Context, id int64) (Startup, error) {
	args := m.Called(ctx, id)
	startup, _ := args.Get(0).(Startup)
	return startup, args.Error(1)
}

func (m *mockStartupRepository) ListStartups(ctx context.Context, limit, offset int) ([]Startup, int64, error) {
	args := m.Called(ctx, limit, offset)
	startups, _ := args.Get(0).([]Startup)
	return startups, args.Get(1).(int64), args.Error(2)
}

func TestStartupService_CreateStartup_DefaultStatus(t *testing.T) {
	repo := new(mockStartupRepository)
	service := NewStartupService(repo)

	repo.On("CreateStartup", mock.Anything, mock.MatchedBy(func(input Startup) bool {
		return input.Status == "failed" && input.Name == "Demo"
	})).Return(Startup{ID: 1, Name: "Demo", Status: "failed"}, nil)

	result, err := service.CreateStartup(context.Background(), Startup{Name: "Demo"})

	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	repo.AssertExpectations(t)
}

func TestStartupService_UpdateStartup_DefaultStatus(t *testing.T) {
	repo := new(mockStartupRepository)
	service := NewStartupService(repo)

	repo.On("UpdateStartup", mock.Anything, mock.MatchedBy(func(input Startup) bool {
		return input.Status == "failed" && input.ID == 10
	})).Return(Startup{ID: 10, Name: "Demo", Status: "failed"}, nil)

	result, err := service.UpdateStartup(context.Background(), Startup{ID: 10, Name: "Demo"})

	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	repo.AssertExpectations(t)
}

// func TestStartupService_ListStartups_Pagination(t *testing.T) {
// 	repo := new(mockStartupRepository)
// 	service := NewStartupService(repo)

// 	repo.On("ListStartups", mock.Anything, 10, 0).Return([]Startup{}, int64(0), nil)

// 	_, _, err := service.ListStartups(context.Background(), 0, 0)

// 	require.NoError(t, err)
// 	repo.AssertExpectations(t)
// }

func TestStartupService_GetStartup_ErrorPropagation(t *testing.T) {
	repo := new(mockStartupRepository)
	service := NewStartupService(repo)

	repo.On("GetStartupByID", mock.Anything, int64(99)).Return(Startup{}, ErrStartupNotFound)

	_, err := service.GetStartupByID(context.Background(), 99)

	require.ErrorIs(t, err, ErrStartupNotFound)
	repo.AssertExpectations(t)
}

func TestStartupService_DeleteStartup_ErrorPropagation(t *testing.T) {
	repo := new(mockStartupRepository)
	service := NewStartupService(repo)

	repo.On("DeleteStartup", mock.Anything, int64(42)).Return(errors.New("boom"))

	err := service.DeleteStartup(context.Background(), 42)

	require.EqualError(t, err, "boom")
	repo.AssertExpectations(t)
}
