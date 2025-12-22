package users

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type UserService interface {
	CreateUser(ctx context.Context, name, email, role, password, profilePicURL, uuid string) (User, error)
	UpdateUser(ctx context.Context, u User) (User, error)
	UpdateUserByUUID(ctx context.Context, currentUUID string, u User) (User, error)
	DeleteUser(ctx context.Context, id int64) error
	DeleteUserByUUID(ctx context.Context, uuid string) error
	GetUserByID(ctx context.Context, id int64) (User, error)
	GetUserByUUID(ctx context.Context, uuid string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	ListUsers(ctx context.Context, page, limit int) ([]User, int64, error)
	Login(ctx context.Context, email, password string) (User, error)
	CheckAndUpdateVerification(ctx context.Context, email string) (bool, error)
}

type userService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(ctx context.Context, name, email, role, password, profilePicURL, uuid string) (User, error) {
	if role != "buyer" && role != "founder" {
		return User{}, errors.New("invalid role")
	}
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	u, err := s.repo.CreateUser(ctx, name, email, role, string(hashBytes), profilePicURL, uuid)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return User{}, errors.New("user exists with that email")
		}
		return User{}, err
	}
	return u, nil
}

func (s *userService) UpdateUser(ctx context.Context, u User) (User, error) {
	if u.Role != "" && u.Role != "buyer" && u.Role != "founder" {
		return User{}, errors.New("invalid role")
	}
	return s.repo.UpdateUser(ctx, u)
}

func (s *userService) UpdateUserByUUID(ctx context.Context, currentUUID string, u User) (User, error) {
	if u.Role != "" && u.Role != "buyer" && u.Role != "founder" {
		return User{}, errors.New("invalid role")
	}

	if u.UUID == "" {
		u.UUID = currentUUID
	}
	return s.repo.UpdateUserByUUID(ctx, currentUUID, u)
}

func (s *userService) DeleteUser(ctx context.Context, id int64) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *userService) DeleteUserByUUID(ctx context.Context, uuid string) error {
	return s.repo.DeleteUserByUUID(ctx, uuid)
}

func (s *userService) GetUserByID(ctx context.Context, id int64) (User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *userService) GetUserByUUID(ctx context.Context, uuid string) (User, error) {
	return s.repo.GetUserByUUID(ctx, uuid)
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (User, error) {
	return s.repo.GetUserByEmail(ctx, email)
}

func (s *userService) ListUsers(ctx context.Context, page, limit int) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	return s.repo.ListUsers(ctx, limit, offset)
}

func (s *userService) Login(ctx context.Context, email, password string) (User, error) {
	id, hash, err := s.repo.GetUserAuthByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return User{}, errors.New("invalid credentials")
		}
		return User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return User{}, errors.New("invalid credentials")
	}

	return s.repo.GetUserByID(ctx, id)
}

func (s *userService) CheckAndUpdateVerification(ctx context.Context, email string) (bool, error) {
	u, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return false, err
	}

	now := time.Now()
	within := false
	if u.VerifiedAt != nil {
		if now.Sub(*u.VerifiedAt) <= 30*24*time.Hour {
			within = true
		}
	}

	if within {
		if err := s.repo.UpdateVerifiedAtByEmail(ctx, email, now); err != nil {
			return false, err
		}
	}

	return within, nil
}
