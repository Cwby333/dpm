package postgres

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"log/slog"

	// "log/slog"

	"github.com/jackc/pgx/v5"
)

type Music struct {
	ID              string `db:"id"`
	Name            string `db:"name"`
	UploaderID      string `db:"uploader_id"`
	Likes           int    `db:"likes"`
	DurationSeconds int    `db:"duration_seconds"`
	Cover *string `db:"music_cover"`
	SongURL *string `db:"song_url"`
}

func MusicPgToMusic(pdb Music) models.Music {
	if pdb.Cover == nil {
		s := ""
		pdb.Cover = &s
	}
	if pdb.SongURL == nil {
		s := ""
		pdb.SongURL = &s
	}

	p := models.Music{
		ID:          pdb.ID,
		Name:        pdb.Name,
		Likes:       pdb.Likes,
		DurationSec: pdb.DurationSeconds,
		UploaderID:  pdb.UploaderID,
		CoverURL: *pdb.Cover,
		SongURL: *pdb.SongURL,
	}

	return p
}

func (p *Postgres) CreateMusic(ctx context.Context, product models.Music) error {
	const op = "./internal/adapters/repo/postgres/music.go.CreateMusic()"

	q := "INSERT INTO music(name, uploader_id, likes, duration_seconds, music_cover, song_url) VALUES ($1, $2, $3, $4, $5, $6)"
	rows, err := p.pool.Query(ctx, q, product.Name, product.UploaderID, product.Likes, product.DurationSec, product.CoverURL, product.SongURL)
	if err != nil {
		return fmt.Errorf("%s INSERT INTO music(): %w", op, err)
	}
	defer rows.Close()

	return nil
}

func (p *Postgres) GetMusic(ctx context.Context, id string) (models.Music, error) {
	const op = "./internal/adapters/repo/postgres/music.go.GetMusic()"

	q := "SELECT id, uploader_id, name, likes, duration_seconds, music_cover, song_url FROM music WHERE id = $1"
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

func (p *Postgres) GetAllMusic(ctx context.Context, u models.User) ([]models.Music, []models.Like, error) {
	const op = "./internal/adapters/repo/postgres/music.go.GetAllMusic()"

	q := "SELECT id, name, uploader_id, likes, duration_seconds, music_cover, song_url FROM music"
	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	products, err := pgx.CollectRows(rows, pgx.RowToStructByName[Music])
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	pSlice := make([]models.Music, 0, 4)
	for i := range products {
		pSlice = append(pSlice, MusicPgToMusic(products[i]))
	}

	if u.ID == "" {
		return pSlice, nil, nil
	}

	q = "SELECT user_id, music_id FROM users_music_likes WHERE user_id = $1"
	rows, err = p.pool.Query(ctx, q, u.ID)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %w", "SELECT music_id FROM users_music_likes WHERE user_id = $1", err.Error()))
		return pSlice, nil, nil
	}	

	l, err := pgx.CollectRows(rows, pgx.RowToStructByName[LikeDB])
	if err != nil {
		slog.Error(err.Error())
		return pSlice, nil, nil
	}

	lSlice := make([]models.Like, 0, len(l))

	for i := range l {
		lSlice = append(lSlice, LDBToLike(l[i]))
	}

	return pSlice, lSlice, nil
}
