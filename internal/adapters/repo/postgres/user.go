package postgres

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	// "log/slog"
)

type UserDB struct {
	ID         string `db:"id"`
	Username   string `db:"username"`
	Email      string `db:"email"`
	HashPsw    string `db:"hash_psw"`
	RegisterAt time.Time `db:"register_at"`
	Likes      int `db:"likes"`
}

func UDBToUser(u UserDB) models.User {
	return models.User{
		ID: u.ID,
		Username: u.Username,
		Email: u.Email,
		HashPsw: u.HashPsw,
		RegisterAt: u.RegisterAt,
		Likes: u.Likes,
	}
}

func (pg *Postgres) CreateUser(ctx context.Context, user models.User) error {
	const op = "./internal/adapters/repo/postgres/user.go.CreateUser()"

	q := "INSERT INTO users(username, hash_psw, email) VALUES ($1, $2, $3) RETURNING id"
	rows, err := pg.pool.Query(ctx, q, user.Username, user.HashPsw, user.Email)
	if err != nil {
		return fmt.Errorf("%s %s: %w", op, q, err)
	}
	defer rows.Close()

	count := ""
	for rows.Next() {
		err = rows.Scan(&count)
		slog.Info(fmt.Sprint(count))
		if err != nil {
			slog.Error(err.Error())
		}
	}
	slog.Info(count)

	if count == "" {
		return fmt.Errorf("%s: %w", op, errors.New("This username or email already exists"))
	}

	return nil
}

func (pg *Postgres) ReadUserID(ctx context.Context, user models.User) (string, error) {
	const op = "./internal/adapters/repo/postgres/user.go.ReadUserID"

	q := "SELECT id FROM users WHERE username = $1"

	rows, err := pg.pool.Query(ctx, q, user.Username)
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", op, q, err)
	}

	id := ""
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return "", fmt.Errorf("%s %s: %w", op, q, err)
		}
	}

	return id, nil
}

func (pg *Postgres) ReadPsw(ctx context.Context, user models.User) (string, error) {
	const op = "./internal/adapters/repo/postgres/user.go.ReadPsw()"

	q := "SELECT hash_psw FROM users WHERE username = $1"

	rows, err := pg.pool.Query(ctx, q, user.Username)
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", op, q, err)
	}

	psw := ""
	for rows.Next() {
		err = rows.Scan(&psw)
		if err != nil {
			return "", fmt.Errorf("%s %s: %w", op, q, err)
		}
	}

	return psw, nil
}

func (pg *Postgres) ReadUser(ctx context.Context, user models.User) (models.User, error) {
	const op = "./internal/adapters/repo/postgres/user.go.ReadUser()"

	q := "SELECT id, username, email, register_at, hash_psw, likes FROM users WHERE id = $1"
	rows, err := pg.pool.Query(ctx, q, user.ID)
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	u, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[UserDB])
	if err != nil {
		return models.User{}, fmt.Errorf("%s: RowToStruct: %w", op, err)
	}

	return UDBToUser(u), nil	
}