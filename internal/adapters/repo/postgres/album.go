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
	LikesCount int `db:"count_likes"`
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
		LikesCount: a.LikesCount,
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
	
	q := "SELECT music_id FROM albums_music WHERE album_id = $1"
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

	q = "DELETE FROM albums_likes WHERE album_id = $1"
	_, err = pg.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
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
	slog.Debug(op, slog.String("albumID", id))

	q := "SELECT name, uploader_id, cover, count_likes FROM albums WHERE id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return models.Album{}, fmt.Errorf("%s: %w", op, err)
	}

	name := ""
	uploaderID := ""
	cover := ""
	likesCount := 0
	for rows.Next() {
		err = rows.Scan(&name, &uploaderID, &cover, &likesCount)
		if err != nil {
			return models.Album{}, fmt.Errorf("%s: %w", op, err)
		}
	}

	return models.Album{
		ID: id,
		Name: name,
		UploaderID: uploaderID,
		Cover: cover,
		LikesCount: likesCount,
	}, nil
}

func (pg *Postgres) GetAlbumsMusic(ctx context.Context, id string) ([]models.LikedTrack, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbumsMusic()"

	q := "SELECT m.id AS music_id, m.name AS music_name, m.uploader_id, u.username, m.likes, m.duration_seconds AS dur_sec, m.song_url, m.music_cover, m.listening_count AS lis_count FROM music m JOIN albums_music am ON m.id = am.music_id JOIN users u ON u.id = m.uploader_id WHERE am.album_id = $1"
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
	slog.Debug(op, slog.String("albumID", id))

	q := "SELECT a.id, a.name, a.uploader_id, a.cover, a.count_likes, u.username FROM albums a JOIN users u ON a.uploader_id = u.id WHERE a.id = $1"
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

func (pg *Postgres) GetAlbumsInfo(ctx context.Context, af models.AlbumFilter) ([]models.AlbumInfo, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetAlbumsInfo()"
	slog.Debug(op)

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Select("a.id, a.name, a.uploader_id, a.cover, a.count_likes, u.username").From("albums a").Join("users u ON a.uploader_id = u.id")

	if af.LikesMin != nil {
		sql = sql.Where("count_likes >= ?", af.LikesMin)
	}

	if af.LikesMax != nil {
		sql = sql.Where("count_likes <= ?", af.LikesMax)
	}

	q, args, err := sql.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := pg.pool.Query(ctx, q, args...)
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
	slog.Debug(op, slog.String("userID", userID))

	q := "SELECT a.id, a.name, a.uploader_id, a.cover, a.count_likes, u.username FROM albums a JOIN users u ON a.uploader_id = u.id WHERE a.uploader_id = $1"
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

func (pg *Postgres) GetUploadedByUserAlbums(ctx context.Context, id string, af models.AlbumFilter) ([]models.Album, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetUploadedByUserAlbums()"
	slog.Debug(op, slog.String("userID", id))

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Select("id, name, cover, count_likes").From("albums").Where("uploader_id = ?", id)

	if af.LikesMin != nil {
		sql = sql.Where("count_likes >= ?", af.LikesMin)
	}

	if af.LikesMax != nil {
		sql = sql.Where("count_likes <= ?", af.LikesMax)
	}

	q, args, err := sql.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := pg.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	aSlice := make([]models.Album, 0)
	aID := ""
	name := ""
	cover := ""
	likesCount := 0
	for rows.Next() {
		err = rows.Scan(&aID, &name, &cover, &likesCount)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		aSlice = append(aSlice, models.Album{
			ID: aID,
			Name: name,
			Cover: cover,
			LikesCount: likesCount,
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
		slog.Info("albumCover: " + album.Cover)

		q := "SELECT music_id FROM albums_music WHERE album_id = $1"
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

	return nil
}

func (pg *Postgres) DeleteLikeAlbum(ctx context.Context, albumID string, userID string) (error) {
	const op = "./internal/adapters/repo/postgres/album.go.DeleteLikeAlbum()"
	slog.Debug(op, slog.String("albumID", albumID), slog.String("userID", userID))

	q := "DELETE FROM albums_likes WHERE album_id = $1 AND user_id = $2"
	_, err := pg.pool.Exec(ctx, q, albumID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q = "UPDATE albums SET count_likes = count_likes - 1 WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, albumID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) LikeAlbum(ctx context.Context, albumID string, userID string) (error) {
	const op = "./internal/adapters/repo/postgres/album.go.LikeAlbum()"
	slog.Debug(op, slog.String("albumID", albumID), slog.String("userID", userID))

	q := "INSERT INTO albums_likes(album_id, user_id) VALUES ($1, $2)"
	_, err := pg.pool.Exec(ctx, q, albumID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q = "UPDATE albums SET count_likes = count_likes + 1 WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, albumID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) GetLikedAlbums(ctx context.Context, userID string, af models.AlbumFilter) ([]models.AlbumInfo, error) {
	const op = "./internal/adapters/repo/postgres/album.go.GetLikedAlbums()"
	slog.Debug(op, slog.String("userID", userID))

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Select("a.id, a.name, a.uploader_id, a.cover, a.count_likes, u.username").From("albums a").Join("albums_likes al ON a.id = al.album_id").Join("users u ON a.uploader_id = u.id").Where("al.user_id = ?", userID)

	if af.LikesMin != nil {
		sql = sql.Where("count_likes >= ?", af.LikesMin)
	}

	if af.LikesMax != nil {
		sql = sql.Where("count_likes <= ?", af.LikesMax)
	}

	q, args, err := sql.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := pg.pool.Query(ctx, q, args...)
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