package postgres

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type LikeDB struct {
	UserID  string `db:"user_id"`
	MusicID string `db:"music_id"`
}

func LDBToLike(l LikeDB) models.Like {
	return models.Like{
		UserID: l.UserID,
		MusicID: l.MusicID,
	}
}

func (pg *Postgres) CreateLike(ctx context.Context, l models.Like) (error) {
	const op = "./internal/adapters/repo/postgres/likes.go.CreateLike()"

	q := "INSERT INTO users_music_likes(user_id, music_id) VALUES ($1, $2)"
	t, err := pg.pool.Exec(ctx, q, l.UserID, l.MusicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if t.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, errors.New("This like exists"))
	}

	q = "UPDATE music SET likes = likes + 1 WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, l.MusicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q = "UPDATE users SET likes = likes + 1 WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, l.UserID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) ReadLikes(ctx context.Context, l models.Like) ([]models.Like, error) {
	const op = "./internal/adapters/repo/postgres/likes.go.ReadLikes()"

	q := "SELECT user_id, music_id FROM likes WHERE user_id = $1"
	rows, err := pg.pool.Query(ctx, q, l.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	li, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[LikeDB])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	likes := make([]models.Like, 0, len(li))
	for i := range li {
		likes = append(likes, LDBToLike(*li[i]))
	}

	return likes, nil
} 

func (pg *Postgres) DeleteLike(ctx context.Context, l models.Like) (error) {
	const op = "./adapters/repo/postgres/likes.go/DeleteLike()"

	q := "DELETE FROM users_music_likes WHERE user_id = $1 AND music_id = $2"
	_, err := pg.pool.Exec(ctx, q, l.UserID, l.MusicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}