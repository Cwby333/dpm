package postgres

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	// "log/slog"

	"github.com/jackc/pgx/v5"
)

type Music struct {
	ID              string `db:"id"`
	Name            string `db:"name"`
	UploaderID      string `db:"uploader_id"`
	Likes           int    `db:"likes"`
	DurationSeconds int    `db:"duration_seconds"`
}

func MusicPgToMusic(pdb Music) models.Music {
	p := models.Music{
		ID:          pdb.ID,
		Name:        pdb.Name,
		Likes:       pdb.Likes,
		DurationSec: pdb.DurationSeconds,
		UploaderID:  pdb.UploaderID,
	}

	return p
}

func (p *Postgres) CreateMusic(ctx context.Context, product models.Music) error {
	const op = "./internal/adapters/repo/postgres/music.go.CreateMusic()"

	q := "INSERT INTO music(name, uploader_id, likes, duration_seconds) VALUES ($1, $2, $3, $4)"
	rows, err := p.pool.Query(ctx, q, product.Name, product.UploaderID, product.Likes, product.DurationSec)
	if err != nil {
		return fmt.Errorf("%s INSERT INTO music(): %w", op, err)
	}
	defer rows.Close()

	return nil
}

func (p *Postgres) GetMusic(ctx context.Context, id string) (models.Music, error) {
	const op = "./internal/adapters/repo/postgres/music.go.GetMusic()"

	q := "SELECT id, uploader_id, name, likes, duration_seconds FROM music WHERE id = $1"
	rows, err := p.pool.Query(ctx, q, id)
	if err != nil {
		return models.Music{}, fmt.Errorf("%s SELECT ... FROM products(): %w", op, err)
	}

	if !rows.Next() {
		return models.Music{}, fmt.Errorf("%s !rows.Next(): %w", op, errors.New("not found by id "+id))
	}

	product, err := pgx.RowToStructByName[Music](rows)
	if err != nil {
		return models.Music{}, fmt.Errorf("%s RowToStructByName(): %w", op, err)
	}

	return MusicPgToMusic(product), nil
}

func (p *Postgres) GetAllMusic(ctx context.Context) ([]models.Music, error) {
	const op = "./internal/adapters/repo/postgres/music.go.GetAllMusic()"

	q := "SELECT id, name, uploader_id, likes, duration_seconds FROM music"
	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	products, err := pgx.CollectRows(rows, pgx.RowToStructByName[Music])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pSlice := make([]models.Music, 0, 4)
	for i := range products {
		pSlice = append(pSlice, MusicPgToMusic(products[i]))
	}

	return pSlice, nil
}
