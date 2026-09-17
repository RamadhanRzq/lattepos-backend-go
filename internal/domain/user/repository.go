package user

import (
	"context"
	"errors"
)

// Error domain yang dipakai lintas layer (service, repository, handler).
var (
	// ErrNotFound dikembalikan saat user tidak ditemukan.
	ErrNotFound = errors.New("user tidak ditemukan")
	// ErrInvalidCredentials dikembalikan saat username/password tidak cocok.
	ErrInvalidCredentials = errors.New("kredensial tidak valid")
	// ErrInvalidInput dikembalikan saat ada field wajib yang kosong.
	ErrInvalidInput = errors.New("semua field wajib diisi")
	// ErrUsernameTaken dikembalikan saat registrasi dengan username yang sudah dipakai.
	ErrUsernameTaken = errors.New("username sudah dipakai")
	// ErrEmailTaken dikembalikan saat registrasi dengan email yang sudah dipakai.
	ErrEmailTaken = errors.New("email sudah dipakai")
)

// UserRepository adalah kontrak penyimpanan user.
// Implementasi: internal/repository (in-memory) dan internal/repository/postgres.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Create(ctx context.Context, u *User) error
}
