package postgres

import (
	"context"
	"dpm/internal/models"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Album struct {
	ID string `db:"id"`
	Name string `db:"name"`
	UploaderID string `db:"uploader_id"`
}

type AlbumInfo struct {
	Album
	UsernameUploader string `db:"username"`
}

func AlbumDBToAlbum(a Album) models.Album {
	return models.Album{
		ID: a.ID,
		Name: a.Name,
		UploaderID: a.UploaderID,
	}
}

func AlbumInfoDBToai(a AlbumInfo) models.AlbumInfo {
	return models.AlbumInfo{
		Album: AlbumDBToAlbum(a.Album),
		Username: a.UsernameUploader,
	}
}

func (pg *Postgres) CreateAlbum(ctx context.Context, album models.Album) (error) {
	const op = "./internal/adapters/repo/postgres/album.go.CreateAlbum()"

	q := "INSERT INTO albums(name, uploader_id) VALUES ($1, $2)"
	_, err := pg.pool.Exec(ctx, q, album.Name, album.UploaderID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) DeleteAlbum(ctx context.Context, id string) (error) {
	const op = "./internal/adapters/repo/postgres/album.go.DeleteAlbum()"

	q := "DELETE FROM albums WHERE id = $1"
	_, err := pg.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) GetAlbum(ctx context.Context, id string) (models.Album, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbum()"

	q := "SELECT name, uploader_id FROM albums WHERE id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return models.Album{}, fmt.Errorf("%s: %w", op, err)
	}

	name := ""
	uploaderID := ""
	for rows.Next() {
		err = rows.Scan(&name, &uploaderID)
		if err != nil {
			return models.Album{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return models.Album{
		ID: id,
		Name: name,
		UploaderID: uploaderID,
	}, nil
}

func (pg *Postgres) GetAlbumsMusic(ctx context.Context, id string) ([]models.Music, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbumsMusic()"

	q := "SELECT m.id, m.name, m.uploader_id, m.likes, m.duration_seconds, m.song_url, m.music_cover FROM music m JOIN albums_music am ON m.id = am.music_id WHERE am.album_id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	musicSlice, err := pgx.CollectRows(rows, pgx.RowToStructByName[Music])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	m := make([]models.Music, 0, len(musicSlice))

	for i := range musicSlice {
		m = append(m, MusicPgToMusic(musicSlice[i]))
	}

	return m, nil
}

func (pg *Postgres) GetAlbumInfo(ctx context.Context, id string) (models.AlbumInfo, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbumInfo()"

	q := "SELECT a.id, a.name, a.uploader_id, u.username FROM albums a JOIN users u ON a.uploader_id = u.id WHERE a.id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return models.AlbumInfo{}, fmt.Errorf("%s: %w", op, err)
	}

	a, err := pgx.RowToStructByName[AlbumInfo](rows)
	if err != nil {
		return models.AlbumInfo{}, fmt.Errorf("%s: %w", op, err)
	}

	return AlbumInfoDBToai(a), nil
}

func (pg *Postgres) GetAlbumsInfo(ctx context.Context) ([]models.AlbumInfo, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbumsInfo()"

	q := "SELECT a.id, a.name, a.uploader_id, u.username FROM albums a JOIN users u ON a.uploader_id = u.id"
	rows, err := pg.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	a, err := pgx.CollectRows(rows, pgx.RowToStructByName[AlbumInfo])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	al := make([]models.AlbumInfo, 0, len(a))
	for i := range a {
		al = append(al, AlbumInfoDBToai(a[i]))
	}

	return al, nil
}