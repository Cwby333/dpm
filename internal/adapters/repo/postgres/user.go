package postgres

import (
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"log/slog"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	// "log/slog"
)

type UserDB struct {
	ID             string    `db:"id"`
	Username       string    `db:"username"`
	Email          string    `db:"email"`
	HashPsw        string    `db:"hash_psw"`
	RegisterAt     time.Time `db:"register_at"`
	Likes          int       `db:"likes"`
	ListeningCount int       `db:"listening_count"`
	FavorCount     int       `db:"favor_count"`
	PrivateProfile bool      `db:"private_profile"`
}

func UDBToUser(u UserDB) models.User {
	return models.User{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		HashPsw:        u.HashPsw,
		RegisterAt:     u.RegisterAt,
		Likes:          u.Likes,
		ListeningCount: u.ListeningCount,
		FavorCount:     u.FavorCount,
		PrivateProfile: u.PrivateProfile,
	}
}

func (pg *Postgres) CreateUser(ctx context.Context, user models.User) error {
	const op = "./internal/adapters/repo/postgres/user.go.CreateUser()"

	q := "INSERT INTO users(username, hash_psw, email, private_profile) VALUES ($1, $2, $3, $4) RETURNING id"
	rows, err := pg.pool.Query(ctx, q, user.Username, user.HashPsw, user.Email, user.PrivateProfile)
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

	q := "SELECT id, username, email, register_at, hash_psw, likes, listening_count, favor_count, private_profile FROM users WHERE id = $1"
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

func (pg *Postgres) GetPublicUsers(ctx context.Context, uf models.UserFilter) ([]models.User, error) {
	const op = "./internal/adapters/repo/postgres/user.go.GetPublicUsers()"

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Select("id, username, register_at, likes, listening_count, favor_count").From("users").Where("private_profile = ?", false)

	if uf.MinRegisterAt != nil {
		sql = sql.Where("register_at >= ?", uf.MinRegisterAt)
	}

	if uf.MaxRegisterAt != nil {
		sql = sql.Where("register_at <= ?", uf.MaxRegisterAt)
	}

	if uf.MinLikes != nil {
		sql = sql.Where("likes >= ?", uf.MinLikes)
	}

	if uf.MaxLikes != nil {
		sql = sql.Where("likes <= ?", uf.MaxLikes)
	}

	if uf.MinLisCount != nil {
		sql = sql.Where("listening_count >= ?", uf.MinLisCount)
	}

	if uf.MaxLisCount != nil {
		sql = sql.Where("listening_count <= ?", uf.MaxLisCount)
	}

	if uf.MinFavorCount != nil {
		sql = sql.Where("favor_count >= ?", uf.MinFavorCount)
	}

	if uf.MaxFavorCount != nil {
		sql = sql.Where("favor_count <= ?", uf.MaxFavorCount)
	}

	q, args, err := sql.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := pg.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", op, q, err)
	}
	defer rows.Close()

	type PublicUserDB struct {
		ID             string    `db:"id"`
		Username       string    `db:"username"`
		RegisterAt     time.Time `db:"register_at"`
		Likes          int       `db:"likes"`
		ListeningCount int       `db:"listening_count"`
		FavorCount     int       `db:"favor_count"`
	}

	users, err := pgx.CollectRows(rows, pgx.RowToStructByName[PublicUserDB])
	if err != nil {
		return nil, fmt.Errorf("%s: RowToStruct: %w", op, err)
	}

	res := make([]models.User, 0, len(users))
	for _, u := range users {
		res = append(res, models.User{
			ID:             u.ID,
			Username:       u.Username,
			RegisterAt:     u.RegisterAt,
			Likes:          u.Likes,
			ListeningCount: u.ListeningCount,
			FavorCount:     u.FavorCount,
		})
	}

	return res, nil
}

func (pg *Postgres) GetUserTracks(ctx context.Context, userID string) ([]models.LikedTrack, error) {
	const op = "./internal/adapters/repo/postgres/user.go.GetUserTracks()"

	q := "SELECT m.id AS music_id, m.name AS music_name, m.music_cover AS music_cover, m.song_url AS song_url, m.uploader_id AS uploader_id, u.username AS username, m.likes AS likes, m.duration_seconds AS dur_sec, m.listening_count AS lis_count FROM music m JOIN users u ON u.id = m.uploader_id WHERE m.uploader_id = $1"
	rows, err := pg.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", op, q, err)
	}

	lt, err := pgx.CollectRows(rows, pgx.RowToStructByName[LikedTrack])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	lSlice := make([]models.LikedTrack, 0, len(lt))
	for i := range lt {
		if lt[i].MusicCover == nil {
			s := ""
			lt[i].MusicCover = &s
		}
		if lt[i].MusicSongURL == nil {
			s := ""
			lt[i].MusicSongURL = &s
		}

		lSlice = append(lSlice, models.LikedTrack{
			MusicID:              lt[i].MusicID,
			MusicName:            lt[i].MusicName,
			MusicCover:           *lt[i].MusicCover,
			MusicSongURL:         *lt[i].MusicSongURL,
			MusicUploaderID:      lt[i].MusicUploaderID,
			UserUsername:         lt[i].UserUsername,
			MusicLikes:           lt[i].MusicLikes,
			MusicDurationSeconds: lt[i].MusicDurationSeconds,
			MusicListeningCount: lt[i].MusicListeningCount,
		})
	}

	return lSlice, nil
}
