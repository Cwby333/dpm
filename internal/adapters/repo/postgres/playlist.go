package postgres

import (
	"context"
	"dpm/internal/models"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type PlaylistDB struct {
	ID         string `db:"id"`
	Name       string `db:"name"`
	UploaderID string `db:"uploader_id"`
	Cover      string `db:"cover"`
	Private    bool   `db:"private"`
	LikesCount int `db:"count_likes"`
}

type PlaylistInfoDB struct {
	ID         string `db:"id"`
	Name       string `db:"name"`
	UploaderID string `db:"uploader_id"`
	Cover      string `db:"cover"`
	Private    bool   `db:"private"`
	Username   string `db:"username"`
	LikesCount int `db:"count_likes"`
}

func PlaylistDBToP(p PlaylistDB) models.Playlist {
	return models.Playlist{
		ID:         p.ID,
		Name:       p.Name,
		Cover:      p.Cover,
		UploaderID: p.UploaderID,
		Private:    p.Private,
		LikesCount: p.LikesCount,
	}
}

func PlaylistInfoDBToPI(p PlaylistInfoDB) models.PlaylistInfo {
	return models.PlaylistInfo{
		Playlist: PlaylistDBToP(PlaylistDB{
			ID:         p.ID,
			Name:       p.Name,
			UploaderID: p.UploaderID,
			Cover:      p.Cover,
			Private:    p.Private,
			LikesCount: p.LikesCount,
		}),
		Username: p.Username,
	}
}

func (pg *Postgres) CreatePlaylist(ctx context.Context, p models.Playlist) error {
	const op = "./internal/adapters/repo/postgres/playlist.go.CreatePlaylist()"

	q := "INSERT INTO playlists(id, name, uploader_id, cover, private) VALUES ($1,$2,$3,$4,$5)"
	_, err := pg.pool.Exec(ctx, q, p.ID, p.Name, p.UploaderID, p.Cover, p.Private)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) GetPlaylist(ctx context.Context, id string) (models.Playlist, error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.GetPlaylist"

	q := "SELECT id, name, uploader_id, cover, private, count_likes FROM playlists WHERE id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return models.Playlist{}, fmt.Errorf("%s: SELECT %w", op, err)
	}

	p, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[PlaylistDB])
	if err != nil {
		return models.Playlist{}, fmt.Errorf("%s: CollectRows %w", op, err)
	}

	return PlaylistDBToP(p), nil
}

func (pg *Postgres) GetUserPlaylists(ctx context.Context, userID string, plf models.PlaylistFilter) ([]models.PlaylistInfo, error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.GetUserPlaylists()"

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Select("p.id, p.name, p.uploader_id, p.cover, p.private, u.username, p.count_likes").From("playlists p").Join("users u ON p.uploader_id = u.id").Where("p.uploader_id = ?", userID)

	slog.Info(fmt.Sprintf("%+v", plf))
	if plf.LikesMin != nil {
		sql = sql.Where("p.count_likes >= ?", plf.LikesMin)
	}

	if plf.LikesMax != nil {
		sql = sql.Where("p.count_likes <= ?", plf.LikesMax)
	}

	q, args, err := sql.ToSql()
	slog.Info(q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := pg.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: SELECT: %w", op, err)
	}

	p, err := pgx.CollectRows(rows, pgx.RowToStructByName[PlaylistInfoDB])
	if err != nil {
		return nil, fmt.Errorf("%s: CollectRows: %w", op, err)
	}

	pl := make([]models.PlaylistInfo, 0, len(p))
	for i := range p {
		pl = append(pl, PlaylistInfoDBToPI(p[i]))
	}

	return pl, nil
}

func (pg *Postgres) GetPublicPlaylists(ctx context.Context, plf models.PlaylistFilter) ([]models.PlaylistInfo, error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.GetPublicPlaylists()"

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Select("p.id, p.name, p.uploader_id, p.cover, p.private, u.username, p.count_likes").From("playlists p").Join("users u ON p.uploader_id = u.id").Where("p.private = ?", false)

	slog.Info(fmt.Sprintf("%+v", plf))
	if plf.LikesMin != nil {
		sql = sql.Where("p.count_likes >= ?", plf.LikesMin)
	}

	if plf.LikesMax != nil {
		sql = sql.Where("p.count_likes <= ?", plf.LikesMax)
	}

	q, args, err := sql.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, &err)
	}
	
	rows, err := pg.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: SELECT: %w", op, err)
	}

	p, err := pgx.CollectRows(rows, pgx.RowToStructByName[PlaylistInfoDB])
	if err != nil {
		return nil, fmt.Errorf("%s: CollectRows: %w", op, err)
	}

	pl := make([]models.PlaylistInfo, 0, len(p))
	for i := range p {
		pl = append(pl, PlaylistInfoDBToPI(p[i]))
	}

	return pl, nil
}

func (pg *Postgres) GetPublicPlaylistsByID(ctx context.Context, id string) ([]models.PlaylistInfo, error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.GetPublicPlaylists()"

	q := "SELECT p.id, p.name, p.uploader_id, p.cover, p.private, u.username, p.count_likes FROM playlists p JOIN users u ON p.uploader_id = u.id WHERE p.private = $1 AND p.uploader_id = $2"
	rows, err := pg.pool.Query(ctx, q, false, id)
	if err != nil {
		return nil, fmt.Errorf("%s: SELECT: %w", op, err)
	}

	p, err := pgx.CollectRows(rows, pgx.RowToStructByName[PlaylistInfoDB])
	if err != nil {
		return nil, fmt.Errorf("%s: CollectRows: %w", op, err)
	}

	pl := make([]models.PlaylistInfo, 0, len(p))
	for i := range p {
		pl = append(pl, PlaylistInfoDBToPI(p[i]))
	}

	return pl, nil
}

func (pg *Postgres) GetPlaylistMusic(ctx context.Context, id string) ([]models.LikedTrack, error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.GetPlaylistMusic"

	q := "SELECT m.id AS music_id, m.name AS music_name, m.uploader_id, u.username, m.likes, m.duration_seconds AS dur_sec, m.song_url, m.music_cover, m.listening_count AS lis_count FROM music m JOIN playlists_music pm ON m.id = pm.music_id JOIN users u ON u.id = m.uploader_id WHERE pm.playlist_id = $1"
	rows, err := pg.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("%s: SELECT %w", op, err)
	}

	m, err := pgx.CollectRows(rows, pgx.RowToStructByName[LikedTrack])
	if err != nil {
		return nil, fmt.Errorf("%s: CollectRows %w", op, err)
	}

	mu := make([]models.LikedTrack, 0, len(m))
	for i := range m {
		mu = append(mu, LikedTrackDBToLT(m[i]))
	}

	return mu, nil
}

func (pg *Postgres) DeletePlaylist(ctx context.Context, id string) error {
	const op = "./internal/adapters/repo/postgres/playlist.go.DeletePlaylist()"


	q := "DELETE FROM playlists_music WHERE playlist_id = $1"
	_, err := pg.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q = "DELETE FROM playlist_likes WHERE playlist_id = $1"
	_, err = pg.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q = "DELETE FROM playlists WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) AddMusicToPlaylist(ctx context.Context, playlistID string, musicID string) error {
	const op = "./internal/adapters/repo/postgres/playlist.go.AddMusicToPlaylist()"

	q := "INSERT INTO playlists_music(playlist_id, music_id) VALUES ($1, $2)"
	_, err := pg.pool.Exec(ctx, q, playlistID, musicID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) UpdatePlaylist(ctx context.Context, playlist models.PlaylistUpdate) (error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.UpdatePlaylist()"

	q := "SELECT name, private, cover FROM playlists WHERE id = $1"
	rows, err := pg.pool.Query(ctx, q, playlist.ID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	name := ""
	cover := ""
	private := false
	for rows.Next() {
		err = rows.Scan(&name, &private, &cover)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	var sql sq.UpdateBuilder
	sql = psql.Update("playlists")
	if playlist.Name != nil && *playlist.Name != "" && *playlist.Name != name {
		sql = sql.Set("name", playlist.Name)
	}

	if playlist.Cover != nil {
		sql = sql.Set("cover", playlist.Cover)
	}

	if playlist.Private != nil {
		sql = sql.Set("private", playlist.Private)
	}

	sql = sql.Where("id = ?", playlist.ID)

	q2, args, err := sql.ToSql()
	if err != nil {
		return fmt.Errorf("%s: Query %w", op, err)
	}
	slog.Info(fmt.Sprint(args))
	slog.Info(q2)

	_, err = pg.pool.Exec(ctx, q2, args...)
	if err != nil {
		return fmt.Errorf("%s: Exec %w", op, err)
	}

	return nil
}

func (pg *Postgres) LikePlaylist(ctx context.Context, playlistID string, userID string) (error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.LikePlaylist"

	q := "INSERT INTO playlist_likes(playlist_id, user_id) VALUES ($1, $2)"
	_, err := pg.pool.Exec(ctx, q, playlistID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q = "UPDATE playlists SET count_likes = count_likes + 1 WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, playlistID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (pg *Postgres) GetLikedPlaylists(ctx context.Context, userID string, plf models.PlaylistFilter) ([]models.PlaylistInfo, error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.GetLikedPlaylists"

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	sql := psql.Select("p.id, p.name, p.uploader_id, p.cover, p.private, u.username, p.count_likes").From("playlists p").Join("playlist_likes pl ON p.id = pl.playlist_id").Join("users u ON p.uploader_id = u.id").Where("pl.user_id = ? AND p.private = ?", userID, false)

	slog.Info(fmt.Sprintf("%+v", plf))
	if plf.LikesMin != nil {
		sql = sql.Where("p.count_likes >= ?", plf.LikesMin)
	}

	if plf.LikesMax != nil {
		sql = sql.Where("p.count_likes <= ?", plf.LikesMax)
	}

	q, args, err := sql.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rows, err := pg.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	pl, err := pgx.CollectRows(rows, pgx.RowToStructByName[PlaylistInfoDB])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	plr := make([]models.PlaylistInfo, 0, len(pl))
	for i := range pl {
		plr = append(plr, PlaylistInfoDBToPI(pl[i]))
	}

	return plr, nil
}

func (pg *Postgres) DeleteLikePlaylist(ctx context.Context, playlistID string, userID string) (error) {
	const op = "./internal/adapters/repo/postgres/playlist.go.DeleteLikePlaylist"

	q := "DELETE FROM playlist_likes WHERE playlist_id = $1 AND user_id = $2"
	_, err := pg.pool.Exec(ctx, q, playlistID, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	q = "UPDATE playlists SET count_likes = count_likes - 1 WHERE id = $1"
	_, err = pg.pool.Exec(ctx, q, playlistID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}