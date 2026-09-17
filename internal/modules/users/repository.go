package users

import "context"

// Repository adalah kontrak penyimpanan user.
// Implementasi: postgres.go.
type Repository interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	// FindByIDInOrg hanya menemukan user yang benar-benar anggota organisasi tersebut.
	FindByIDInOrg(ctx context.Context, orgID, id string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	// ListByOrg hanya mengembalikan user yang menjadi anggota organisasi tersebut.
	ListByOrg(ctx context.Context, orgID string) ([]*User, error)
	Create(ctx context.Context, u *User) error
}
