package postgres

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type Album struct {
	ID string `db:"id"`
	Name string `db:"name"`
	UploaderID string `db:"uploader_id"`
	Cover string `db:"cover"`
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
		Cover: a.Cover,
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

	q := "INSERT INTO albums(id, name, uploader_id, cover) VALUES ($1, $2, $3, $4)"
	_, err := pg.pool.Exec(ctx, q, album.ID, album.Name, album.UploaderID, album.Cover)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) AddMusicToAlbum(ctx context.Context, albumID string, musicID string) error {
	const op = "./internal/adapters/repo/postgres/album.go.AddMusicToAlbum()"

	q := "INSERT INTO albums_music(album_id, music_id) VALUES ($1, $2)"
	_, err := pg.pool.Exec(ctx, q, albumID, musicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) DeleteAlbum(ctx context.Context, id string) (error) {
	const op = "./internal/adapters/repo/postgres/album.go.DeleteAlbum()"
	
	q := "SELECT music_id FROM album_music WHERE album_id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: SELECT music_id ... %w", op, err)
	}

	musicsIDS := make([]string, 0)
	for rows.Next() {
		musicID := ""
		err = rows.Scan(&musicID)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		musicsIDS = append(musicsIDS, musicID)
	}

	q = "DELETE FROM albums_music WHERE album_id = $1"
	_, err = pg.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: DELETE FROM albums_music ... %w", op, err)
	}

	q = "DELETE FROM music WHERE id = $1"
	for i := range musicsIDS {
		_, err = pg.pool.Exec(ctx, q, musicsIDS[i])
		if err != nil {
			return fmt.Errorf("%s: DELETE FROM music %w", op, err)
		}
	}

	q = "DELETE FROM albums WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) GetAlbum(ctx context.Context, id string) (models.Album, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbum()"

	q := "SELECT name, uploader_id, cover FROM albums WHERE id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return models.Album{}, fmt.Errorf("%s: %w", op, err)
	}

	name := ""
	uploaderID := ""
	cover := ""
	for rows.Next() {
		err = rows.Scan(&name, &uploaderID, &cover)
		if err != nil {
			return models.Album{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return models.Album{
		ID: id,
		Name: name,
		UploaderID: uploaderID,
		Cover: cover,
	}, nil
}

func (pg *Postgres) GetAlbumsMusic(ctx context.Context, id string) ([]models.LikedTrack, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbumsMusic()"

	q := "SELECT m.id AS music_id, m.name AS music_name, m.uploader_id, u.username, m.likes, m.duration_seconds AS dur_sec, m.song_url, m.music_cover FROM music m JOIN albums_music am ON m.id = am.music_id JOIN users u ON u.id = m.uploader_id WHERE am.album_id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	musicSlice, err := pgx.CollectRows(rows, pgx.RowToStructByName[LikedTrack])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	m := make([]models.LikedTrack, 0, len(musicSlice))

	for i := range musicSlice {
		m = append(m, LikedTrackDBToLT(musicSlice[i]))
	}

	return m, nil
}

func (pg *Postgres) GetAlbumInfo(ctx context.Context, id string) (models.AlbumInfo, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbumInfo()"

	q := "SELECT a.id, a.name, a.uploader_id, a.cover, u.username FROM albums a JOIN users u ON a.uploader_id = u.id WHERE a.id = $1"
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

	q := "SELECT a.id, a.name, a.uploader_id, a.cover, u.username FROM albums a JOIN users u ON a.uploader_id = u.id"
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

func (pg *Postgres) GetUserAlbums(ctx context.Context, userID string) ([]models.AlbumInfo, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetUserAlbums()"

	q := "SELECT a.id, a.name, a.uploader_id, a.cover, u.username FROM albums a JOIN users u ON a.uploader_id = u.id WHERE a.uploader_id = $1"
	rows, err := pg.pool.Query(ctx, q, userID)
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

func (pg *Postgres) GetUploadedByUserAlbums(ctx context.Context, id string) ([]models.Album, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetUploadedByUserAlbums()"

	q := "SELECT id, name, cover FROM albums WHERE uploader_id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	aSlice := make([]models.Album, 0)
	aID := ""
	name := ""
	cover := ""
	for rows.Next() {
		err = rows.Scan(&aID, &name, &cover)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		aSlice = append(aSlice, models.Album{
			ID: aID,
			Name: name,
			Cover: cover,
		})
	}

	return aSlice, nil
}

func (pg *Postgres) UpdateAlbum(ctx context.Context, album models.Album) (error) {
	const op = "./internal/adapters/repo/postgres/album.go.UpdateAlbum()"

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Update("albums")

	if album.Name != "" {
		sql = sql.Set("name", album.Name)
	}

	if album.Cover != "" {
		sql = sql.Set("cover", album.Cover)
	}

	sql.Where("id = ?", album.ID)

	q, args, err := sql.ToSql()
	if err != nil {
		return fmt.Errorf("%s: ToSql %w", op, err)
	}

	slog.Info("UpdateAlbum sql " + q)

	_, err = pg.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("%s: EXEC %w", op, err)
	}

	q = "SELECT music_id FROM albums_music WHERE album_id = $1"
	rows, err := pg.pool.Query(ctx, q, album.ID)
	if err != nil {
		return fmt.Errorf("%s: SELECT music_id ... %w", op, err)
	}

	musicsIDS := make([]string, 0)
	for rows.Next() {
		musicID := ""
		err = rows.Scan(&musicID)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		musicsIDS = append(musicsIDS, musicID)
	}
	
	q = "UPDATE music SET music_cover = $1 WHERE id = $2"
	for i := range musicsIDS {
		_, err = pg.pool.Exec(ctx, q, album.Cover, musicsIDS[i])
		if err != nil {
			return fmt.Errorf("%s: UPDATE music SET cover ... %w", op, err)
		}
	}

	return nil
}