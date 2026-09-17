package users

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Service menangani logika bisnis user: listing, lookup, dan registrasi.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]User, len(list))
	for i, u := range list {
		out[i] = *u
	}

	return out, nil
}

// ListByOrg mengembalikan user yang menjadi anggota organisasi tersebut.
func (s *Service) ListByOrg(ctx context.Context, orgID string) ([]User, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, ErrInvalidInput
	}

	list, err := s.repo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	out := make([]User, len(list))
	for i, u := range list {
		out[i] = *u
	}

	return out, nil
}

// GetByIDInOrg hanya mengembalikan user yang merupakan anggota organisasi tersebut.
func (s *Service) GetByIDInOrg(ctx context.Context, orgID, id string) (User, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return User{}, ErrInvalidInput
	}

	u, err := s.repo.FindByIDInOrg(ctx, orgID, id)
	if err != nil {
		return User{}, err
	}
	return *u, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	return *u, nil
}

// Create mendaftarkan user baru dengan password yang di-hash bcrypt.
func (s *Service) Create(ctx context.Context, username, name, email, password string) (User, error) {
	username = strings.TrimSpace(strings.ToLower(username))
	email = strings.TrimSpace(strings.ToLower(email))

	if username == "" || name == "" || email == "" || password == "" {
		return User{}, ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	u := &User{
		Username:     username,
		Name:         name,
		Email:        email,
		Role:         "cashier", // sama dengan default kolom role di migration
		PasswordHash: string(hash),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return User{}, err
	}

	return *u, nil
}
