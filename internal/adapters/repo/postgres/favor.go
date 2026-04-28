package postgres

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type FavorResponseDB struct {
	MusicID              string  `db:"music_id"`
	MusicName            string  `db:"music_name"`
	MusicCover           *string `db:"music_cover"`
	MusicSongURL         *string `db:"song_url"`
	MusicUploaderID      string  `db:"uploader_id"`
	UserUsername         string  `db:"username"`
	MusicLikes           int     `db:"likes"`
	MusicDurationSeconds int     `db:"dur_sec"`
}

func FavorDBToModel(lhdb FavorResponseDB) models.ListeningHistoryResponse {
	if lhdb.MusicCover == nil {
		s := ""
		lhdb.MusicCover = &s
	}
	if lhdb.MusicSongURL == nil {
		s := ""
		lhdb.MusicSongURL = &s
	}

	return models.ListeningHistoryResponse{
		MusicID:              lhdb.MusicID,
		MusicName:            lhdb.MusicName,
		MusicCover:           *lhdb.MusicCover,
		MusicSongURL:         *lhdb.MusicSongURL,
		MusicUploaderID:      lhdb.MusicUploaderID,
		UserUsername:         lhdb.UserUsername,
		MusicLikes:           lhdb.MusicLikes,
		MusicDurationSeconds: lhdb.MusicDurationSeconds,
	}
}

func (p *Postgres) CreateFavor(ctx context.Context, listeningHistoryItem models.ListeningHistory) error {
	const op = "./internal/adapters/repo/postgres/favor.go.CreateFavor()"

	q := "INSERT INTO favor(user_id, music_id) VALUES ($1, $2)"
	tag, err := p.pool.Exec(ctx, q, listeningHistoryItem.UserID, listeningHistoryItem.MusicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		slog.Info("Rows affected by CreateListeningHistoryItem 0")
	}

	q = "UPDATE users SET favor_count = favor_count + 1 WHERE id = $1"
	_, err = p.pool.Exec(ctx, q, listeningHistoryItem.UserID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (p *Postgres) DeleteFavor(ctx context.Context, lhi models.ListeningHistory) error {
	const op = "./internal/adapters/repo/postgres/favor.go.DeleteFavor()"

	q := "DELETE FROM favor WHERE user_id = $1 AND music_id = $2"
	tag, err := p.pool.Exec(ctx, q, lhi.UserID, lhi.MusicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		slog.Info("Rows affected by DeletLHI 0")
	}

	q = "UPDATE users SET favor_count = favor_count - 1 WHERE id = $1"
	_, err = p.pool.Exec(ctx, q, lhi.UserID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (p *Postgres) ReadFavor(ctx context.Context, lhi models.ListeningHistory) ([]models.ListeningHistoryResponse, error) {
	const op = "./internal/adapters/repo/postgres/favor.go.ReadFavor()"

	q := "SELECT m.id AS music_id, m.name AS music_name, m.music_cover AS music_cover, m.song_url AS song_url, m.uploader_id AS uploader_id, u.username AS username, m.likes AS likes, m.duration_seconds AS dur_sec FROM music m JOIN favor lh ON m.id = lh.music_id JOIN users u ON u.id = lh.user_id WHERE lh.user_id = $1"
	rows, err := p.pool.Query(ctx, q, lhi.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: Query: %w", op, err)
	}

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[FavorResponseDB])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lhr := make([]models.ListeningHistoryResponse, 0, len(slice))

	for i := range slice {
		lhr = append(lhr, FavorDBToModel(slice[i]))
	}

	return lhr, nil
}
