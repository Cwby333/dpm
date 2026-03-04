package postgres

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"log/slog"
	// "log/slog"
)

func (pg *Postgres) CreateUser(ctx context.Context, user models.User) (error) {
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