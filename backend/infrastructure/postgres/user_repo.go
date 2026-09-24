package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/RakhaYandra/pulse/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepo struct {
	DB *sql.DB
}

func (r UserRepo) Create(ctx context.Context, u domain.User) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO users(id,email,password_hash,name) VALUES($1,$2,$3,$4)`,
		u.ID, u.Email, u.PasswordHash, u.Name)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (r UserRepo) ByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := r.DB.QueryRowContext(ctx, `SELECT id,email,password_hash,name,created_at FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.User{}, domain.ErrNotFound
	}
	return u, err
}

func (r UserRepo) ByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := r.DB.QueryRowContext(ctx, `SELECT id,email,password_hash,name,created_at FROM users WHERE id=$1`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return domain.User{}, domain.ErrNotFound
	}
	return u, err
}
