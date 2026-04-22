package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/arcgolabs/dix/examples/backend/domain"
	"github.com/arcgolabs/dix/examples/backend/event"
	"github.com/arcgolabs/dix/examples/backend/repo"
)

// UserService implements the user operations exposed by the backend example.
type UserService interface {
	List(ctx context.Context, search string, limit, offset int) ([]domain.User, int, error)
	Get(ctx context.Context, id int64) (domain.User, bool, error)
	Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error)
	Update(ctx context.Context, id int64, in domain.UpdateUserInput) (domain.User, bool, error)
	Delete(ctx context.Context, id int64) (bool, error)
}

type userService struct {
	repository repo.UserRepository
	bus        eventx.BusRuntime
	log        *slog.Logger
}

// NewUserService creates a user service backed by the repository and event bus.
func NewUserService(userRepo repo.UserRepository, bus eventx.BusRuntime, log *slog.Logger) UserService {
	return &userService{repository: userRepo, bus: bus, log: log}
}

func (s *userService) List(ctx context.Context, search string, limit, offset int) ([]domain.User, int, error) {
	users, total, err := s.repository.List(ctx, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

func (s *userService) Get(ctx context.Context, id int64) (domain.User, bool, error) {
	user, ok, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, false, fmt.Errorf("get user: %w", err)
	}
	return user, ok, nil
}

func (s *userService) Create(ctx context.Context, in domain.CreateUserInput) (domain.User, error) {
	user, err := s.repository.Create(ctx, in)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user: %w", err)
	}
	if err := s.bus.PublishAsync(ctx, event.UserCreatedEvent{
		UserID: user.ID, UserName: user.Name, Email: user.Email, CreatedAt: user.CreatedAt,
	}); err != nil && s.log != nil {
		s.log.Warn("publish user created event failed",
			slog.String("error", err.Error()),
			slog.Int64("user_id", user.ID),
		)
	}
	return user, nil
}

func (s *userService) Update(ctx context.Context, id int64, in domain.UpdateUserInput) (domain.User, bool, error) {
	user, ok, err := s.repository.Update(ctx, id, in)
	if err != nil {
		return domain.User{}, false, fmt.Errorf("update user: %w", err)
	}
	return user, ok, nil
}

func (s *userService) Delete(ctx context.Context, id int64) (bool, error) {
	deleted, err := s.repository.Delete(ctx, id)
	if err != nil {
		return false, fmt.Errorf("delete user: %w", err)
	}
	return deleted, nil
}
