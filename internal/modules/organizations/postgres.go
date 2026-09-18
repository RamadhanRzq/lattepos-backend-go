package organizations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/ramadhanrzq/backend-go/internal/modules/users"
)

type postgresRepository struct {
	db *sql.DB

	stmtCreate       *sql.Stmt
	stmtFindByID     *sql.Stmt
	stmtFindBySlug   *sql.Stmt
	stmtList         *sql.Stmt
	stmtAddMember    *sql.Stmt
	stmtRemoveMember *sql.Stmt
	stmtIsMember     *sql.Stmt
	stmtListMembers  *sql.Stmt
	stmtListByUser   *sql.Stmt
}

var _ Repository = (*postgresRepository)(nil)

func NewRepository(db *sql.DB) (Repository, error) {
	stmtCreate, err := db.Prepare(`
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id, created_at`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtCreate: %w", err)
	}

	stmtFindByID, err := db.Prepare(`
		SELECT id, name, slug, created_at
		FROM organizations
		WHERE id = $1`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtFindByID: %w", err)
	}

	stmtFindBySlug, err := db.Prepare(`
		SELECT id, name, slug, created_at
		FROM organizations
		WHERE lower(slug) = lower($1)`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtFindBySlug: %w", err)
	}

	stmtList, err := db.Prepare(`
		SELECT id, name, slug, created_at
		FROM organizations
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtList: %w", err)
	}

	stmtAddMember, err := db.Prepare(`
		INSERT INTO organization_members (org_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (org_id, user_id) DO NOTHING`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtAddMember: %w", err)
	}

	stmtRemoveMember, err := db.Prepare(`
		DELETE FROM organization_members
		WHERE org_id = $1 AND user_id = $2`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtRemoveMember: %w", err)
	}

	stmtIsMember, err := db.Prepare(`
		SELECT EXISTS(
			SELECT 1 FROM organization_members
			WHERE org_id = $1 AND user_id = $2
		)`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtIsMember: %w", err)
	}
	stmtListMembers, err := db.Prepare(`
		SELECT u.id, u.username, u.name, u.email, u.role, u.created_at, u.updated_at
		FROM users u
		JOIN organization_members om ON om.user_id = u.id
		WHERE om.org_id = $1
		ORDER BY u.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtListMembers: %w", err)
	}

	stmtListByUser, err := db.Prepare(`
		SELECT o.id, o.name, o.slug, o.created_at
		FROM organizations o
		JOIN organization_members om ON om.org_id = o.id
		WHERE om.user_id = $1
		ORDER BY o.created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("prepare stmtListByUser: %w", err)
	}

	return &postgresRepository{
		db:               db,
		stmtCreate:       stmtCreate,
		stmtFindByID:     stmtFindByID,
		stmtFindBySlug:   stmtFindBySlug,
		stmtList:         stmtList,
		stmtAddMember:    stmtAddMember,
		stmtRemoveMember: stmtRemoveMember,
		stmtIsMember:     stmtIsMember,
		stmtListMembers:  stmtListMembers,
		stmtListByUser:   stmtListByUser,
	}, nil
}

func (r *postgresRepository) Create(ctx context.Context, org *Organization) error {
	err := r.stmtCreate.QueryRowContext(ctx, org.Name, org.Slug).
		Scan(&org.ID, &org.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrSlugTaken
		}
		return fmt.Errorf("postgres: create organization: %w", err)
	}
	return nil
}

func (r *postgresRepository) FindByID(ctx context.Context, id string) (*Organization, error) {
	var org Organization
	err := r.stmtFindByID.QueryRowContext(ctx, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find org by id: %w", err)
	}
	return &org, nil
}

func (r *postgresRepository) FindBySlug(ctx context.Context, slug string) (*Organization, error) {
	var org Organization
	err := r.stmtFindBySlug.QueryRowContext(ctx, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("postgres: find org by slug: %w", err)
	}
	return &org, nil
}

func (r *postgresRepository) List(ctx context.Context) ([]Organization, error) {
	rows, err := r.stmtList.QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres: list organizations: %w", err)
	}
	defer rows.Close()

	var list []Organization
	for rows.Next() {
		var org Organization
		if err := rows.Scan(&org.ID, &org.Name, &org.Slug, &org.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan organization: %w", err)
		}
		list = append(list, org)
	}
	return list, rows.Err()
}

func (r *postgresRepository) AddMember(ctx context.Context, orgID, userID string) error {
	_, err := r.stmtAddMember.ExecContext(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("postgres: add org member: %w", err)
	}
	return nil
}

func (r *postgresRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	_, err := r.stmtRemoveMember.ExecContext(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("postgres: remove org member: %w", err)
	}
	return nil
}

func (r *postgresRepository) IsMember(ctx context.Context, orgID, userID string) (bool, error) {
	var exists bool
	err := r.stmtIsMember.QueryRowContext(ctx, orgID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("postgres: is member check: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) ListMembers(ctx context.Context, orgID string) ([]users.User, error) {
	rows, err := r.stmtListMembers.QueryContext(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list org members: %w", err)
	}
	defer rows.Close()

	var members []users.User
	for rows.Next() {
		var u users.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Name, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan member user: %w", err)
		}
		members = append(members, u)
	}
	return members, rows.Err()
}

func (r *postgresRepository) ListByUserID(ctx context.Context, userID string) ([]Organization, error) {
	rows, err := r.stmtListByUser.QueryContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list user organizations: %w", err)
	}
	defer rows.Close()

	var orgs []Organization
	for rows.Next() {
		var o Organization
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan user organization: %w", err)
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}
