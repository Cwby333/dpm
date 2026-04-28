package postgres

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
)

type ListeningHistoryResponseDB struct {
	MusicID              string    `db:"music_id"`
	MusicName            string    `db:"music_name"`
	MusicCover           *string   `db:"music_cover"`
	MusicSongURL         *string   `db:"song_url"`
	MusicUploaderID      string    `db:"uploader_id"`
	UserUsername         string    `db:"username"`
	MusicLikes           int       `db:"likes"`
	MusicDurationSeconds int       `db:"dur_sec"`
	ListeningDate        time.Time `db:"lis_date"`
}

func ListeningHistoryDBToModel(lhdb ListeningHistoryResponseDB) models.ListeningHistoryResponse {
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
		ListeningDate:        lhdb.ListeningDate,
	}
}

func (p *Postgres) CreateListeningHistoryItem(ctx context.Context, listeningHistoryItem models.ListeningHistory) error {
	const op = "./internal/adapters/repo/postgres/listeningHistory.go.CreateListeningHistoryItem()"

	q := "INSERT INTO listening_history(user_id, music_id) VALUES ($1, $2)"
	tag, err := p.pool.Exec(ctx, q, listeningHistoryItem.UserID, listeningHistoryItem.MusicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		slog.Info("Rows affected by CreateListeningHistoryItem 0")
	}

	q = "UPDATE users SET listening_count = listening_count + 1 WHERE id = $1"
	_, err = p.pool.Exec(ctx, q, listeningHistoryItem.UserID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (p *Postgres) DeleteListeningHistoryItem(ctx context.Context, lhi models.ListeningHistory) error {
	const op = "./internal/adapters/repo/postgres/listeningHistory.go.DeleteListeningHistoryItem()"

	q := "DELETE FROM listening_history WHERE user_id = $1 AND music_id = $2  AND listening_date = $3"
	tag, err := p.pool.Exec(ctx, q, lhi.UserID, lhi.MusicID, lhi.ListeningDate)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		slog.Info("Rows affected by DeletLHI 0")
	}

	q = "UPDATE users SET listening_count = listening_count - 1 WHERE id = $1"
	_, err = p.pool.Exec(ctx, q, lhi.UserID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (p *Postgres) ReadListeningHistory(ctx context.Context, lhi models.ListeningHistory) ([]models.ListeningHistoryResponse, error) {
	const op = "./internal/adapters/repo/postgres/listeningHistory.go.ReadListeningHistory()"

	q := "SELECT m.id AS music_id, m.name AS music_name, m.music_cover AS music_cover, m.song_url AS song_url, m.uploader_id AS uploader_id, u.username AS username, m.likes AS likes, m.duration_seconds AS dur_sec, lh.listening_date AS lis_date FROM music m JOIN listening_history lh ON m.id = lh.music_id JOIN users u ON u.id = lh.user_id WHERE lh.user_id = $1"
	rows, err := p.pool.Query(ctx, q, lhi.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: Query: %w", op, err)
	}

	slice, err := pgx.CollectRows(rows, pgx.RowToStructByName[ListeningHistoryResponseDB])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lhr := make([]models.ListeningHistoryResponse, 0, len(slice))

	for i := range slice {
		lhr = append(lhr, ListeningHistoryDBToModel(slice[i]))
	}

	return lhr, nil
}
