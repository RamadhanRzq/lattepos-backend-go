package users

import (
	"errors"
	"time"
)

// Error domain yang dipakai lintas layer di module users.
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

// User adalah entity domain user.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"` // tidak pernah ikut di-response
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
