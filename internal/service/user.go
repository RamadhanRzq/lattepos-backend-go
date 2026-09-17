package service

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/ramadhanrzq/backend-go/internal/domain/user"
)

// UserService menangani logika bisnis user: listing, lookup, dan registrasi.
type UserService struct {
	users user.UserRepository
}

func NewUserService(users user.UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) List(ctx context.Context) ([]user.User, error) {
	users, err := s.users.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]user.User, len(users))
	for i, u := range users {
		out[i] = *u
	}

	return out, nil
}

func (s *UserService) GetByID(ctx context.Context, id string) (user.User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		return user.User{}, err
	}
	return *u, nil
}

// Create mendaftarkan user baru dengan password yang di-hash bcrypt.
func (s *UserService) Create(ctx context.Context, username, name, email, password string) (user.User, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	email = strings.TrimSpace(strings.ToLower(email))

	if username == "" || name == "" || email == "" || password == "" {
		return user.User{}, user.ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, err
	}

	u := &user.User{
		Username:     username,
		Name:         name,
		Email:        email,
		Role:         "cashier", // sama dengan default kolom role di migration
		PasswordHash: string(hash),
	}

	if err := s.users.Create(ctx, u); err != nil {
		return user.User{}, err
	}

	return *u, nil
}
