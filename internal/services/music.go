package services

import (
	"bytes"
	"context"
	"dpm/internal/models"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	songPostfix      = "-song"
	songImagePostfix = "-songImage"
)

type RepoMusic interface {
	CreateMusic(ctx context.Context, product models.Music) error
	GetMusic(ctx context.Context, id string, userID string) (models.Music, models.Like, error)
	GetAllMusic(ctx context.Context, u models.User) ([]models.Music, []models.Like, error)
	GetMusicSQ(ctx context.Context, m models.MusicFilterQuery, userID string) ([]models.Music, []models.Like, error)
	GetMusicByUploaderID(ctx context.Context, id string) ([]models.Music, error)
	AddListening(ctx context.Context, id string) (error)
}

type S3 interface {
	UploadObject(ctx context.Context, key string, data []byte, contentType string) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, key string) error
	GetPresignURL(ctx context.Context, id string) (string, error)
	ListObjects(ctx context.Context, prefix string, suf string) ([]string, error)
}

type MusicService struct {
	repo RepoMusic
	s3   S3
}

func NewMusicService(repo RepoMusic, s3 S3) *MusicService {
	return &MusicService{
		repo: repo,
		s3:   s3,
	}
}

func (s *MusicService) CreateMusic(ctx context.Context, songID string, product models.Music) error {
	const op = "./internal/services/music.go.CreateMusic()"

	slog.Info(fmt.Sprintf("%+v", product))

	err := s.repo.CreateMusic(ctx, product)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *MusicService) GetMusic(ctx context.Context, id string, userID string) (models.Music, models.Like, error) {
	const op = "./internal/services/music.go.GetMusic()"

	product, like, err := s.repo.GetMusic(ctx, id, userID)
	if err != nil {
		slog.Info(err.Error())
		return models.Music{}, models.Like{}, fmt.Errorf("%s: %w", op, err)
	}

	return product, like, nil
}

func (s *MusicService) GetAllMusic(ctx context.Context, u models.User) ([]models.Music, []models.Like, error) {
	const op = "./internal/services/music.go.GetAllProducts()"

	p, l, err := s.repo.GetAllMusic(ctx, u)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	return p, l, nil
}

// func (s *MusicService) UploadSong(ctx context.Context, data []byte, contentType string) (string, error) {
// 	const op = "./internal/services/music.go.UploadMusic()"

// 	songID := uuid.NewString()

// 	err := s.s3.UploadObject(ctx, songID, data, contentType)
// 	if err != nil {
// 		return "", fmt.Errorf("%s: %w", op, err)
// 	}

// 	return songID, nil
// }

// func (s *MusicService) UploadMusicCover(ctx context.Context, key string, data []byte, contentType string) error {
// 	const op = "./internal/services/music.go.UploadMusicCover()"

// 	err := s.s3.UploadObject(ctx, key, data, contentType)
// 	if err != nil {
// 		return fmt.Errorf("%s: %w", op, err)
// 	}

// 	return nil
// }

func (s *MusicService) UploadMusic(ctx context.Context, musicData map[string]models.DataAndCT, music models.Music) error {
	const op = "./internal/services/music.go.UploadSong()"

	musicID := uuid.NewString()

	songData, ok := musicData["songData"]
	if !ok {
		return fmt.Errorf("%s: %w", op, errors.New("Missing songData, upload is unreally"))
	}
	slog.Info(fmt.Sprint("UploadSong: songData:", fmt.Sprintf("%v, %v, %v", songData.Name, songData.ContentType, songData.Data[:100])))
	slog.Info("songDataSize", slog.Int("size", len(songData.Data)))

	songID := musicID + songPostfix
	err := s.s3.UploadObject(ctx, songID, songData.Data, songData.ContentType)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	coverData := musicData["coverData"]
	slog.Info(fmt.Sprintf("cover data size: %v", len(coverData.Data)))
	coverID := musicID + songImagePostfix
	err = s.s3.UploadObject(ctx, coverID, coverData.Data, songData.ContentType)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	music.SongURL = songID
	music.CoverURL = coverID
	music.ID = musicID
	slog.Info(fmt.Sprintf("coverURL, songURL: %v, %v", music.CoverURL, music.SongURL))
	err = s.CreateMusic(ctx, "", music)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func convertToHLS(mp3Path, tmpDir string) error {
	probeCmd := exec.Command("ffprobe", 
		"-v", "error", 
		"-select_streams", "a:0", 
		"-show_entries", "stream=bitrate", 
		"-of", "default=noprint_wrappers=1", 
		mp3Path,
	)
	
	var probeOut, probeErr bytes.Buffer
	probeCmd.Stdout = &probeOut
	probeCmd.Stderr = &probeErr
	
	if err := probeCmd.Run(); err != nil {
		slog.Error("ffprobe stderr", slog.String("stderr", probeErr.String()))
		return fmt.Errorf("ошибка ffprobe: %w", err)
	}

	output := strings.TrimSpace(probeOut.String())
	bitrate := "256k"

	if strings.HasPrefix(output, "bitrate=") {
		rawBitrate := strings.TrimPrefix(output, "bitrate=")
		if rawBitrate != "N/A" && rawBitrate != "" {
			var b int
			_, err := fmt.Sscanf(rawBitrate, "%d", &b)
			if err == nil && b > 0 {
				bitrate = fmt.Sprintf("%dk", b/1000)
			}
		}
	}

	var ffmpegErr bytes.Buffer
	cmd := exec.Command("ffmpeg", "-i", mp3Path,
		"-vn",
		"-c:a", "aac", 
		"-b:a", bitrate,
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-hls_segment_filename", filepath.Join(tmpDir, "seg%03d.ts"),
		filepath.Join(tmpDir, "playlist.m3u8"),
	)
	cmd.Stderr = &ffmpegErr

	if err := cmd.Run(); err != nil {
		slog.Error("ffmpeg stderr", slog.String("stderr", ffmpegErr.String()))
		return fmt.Errorf("ошибка ffmpeg: %w", err)
	}

	return nil
}

func (s *MusicService) UploadHLSMusic(ctx context.Context, mData map[string]models.DataAndCT, m models.Music) (error) {
	const op = "./internal/services/music.go.UploadMusicHLS"

	musicID := uuid.NewString()

	songData, ok := mData["songData"]
	if !ok {
		return fmt.Errorf("%s: %w", op, errors.New("Missing songData, upload is unreally"))
	}
	slog.Info(fmt.Sprint("UploadSong: songData:", fmt.Sprintf("%v, %v, %v", songData.Name, songData.ContentType, songData.Data[:100])))
	slog.Info("songDataSize", slog.Int("size", len(songData.Data)))

	songID := musicID

	slog.Debug("UploadHLS", slog.String("songID", songID))

	tmpDir, err := os.MkdirTemp("", musicID)
	if err != nil {
		return fmt.Errorf("%s: MkdirTemp %w", op, err)
	}
	defer os.RemoveAll(tmpDir)

	mp3Path := filepath.Join(tmpDir, "input.mp3")
	err = os.WriteFile(mp3Path, songData.Data, 0644)
	if err != nil {
		return fmt.Errorf("%s: WriteTempFile %w", op, err)
	}

	cmd := convertToHLS(mp3Path, tmpDir)

	if cmd != nil {
		return fmt.Errorf("%s: ffmpeg exec %s", op, cmd.Error())
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("%s: ReadTmpDir %w", op, err)
	}

	for _, f := range files {
		slog.Debug("UploadHLSMusic", slog.String("fName", f.Name()))

		data, err := os.ReadFile(filepath.Join(tmpDir, f.Name()))
		if err != nil {
			return fmt.Errorf("%s: ReadFile %w", op, err)
		}
		
		ct := "audio/mpeg"
		if strings.HasSuffix(f.Name(), "m3u8") {
			ct = "application/vnd.apple.mpegurl"
		}
		
		err = s.s3.UploadObject(ctx, musicID + "-hls/" + f.Name(), data, ct)
		if err != nil {
			return fmt.Errorf("%s: UploadObject %w", op, err)
		}
	}

	coverData := mData["coverData"]
	slog.Info(fmt.Sprintf("cover data size: %v", len(coverData.Data)))
	coverID := musicID + songImagePostfix
	err = s.s3.UploadObject(ctx, coverID, coverData.Data, songData.ContentType)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	m.SongURL = songID + "-hls/playlist.m3u8"
	m.CoverURL = coverID
	m.ID = musicID
	slog.Info(fmt.Sprintf("coverURL, songURL: %v, %v", m.CoverURL, m.SongURL))
	err = s.CreateMusic(ctx, "", m)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *MusicService) GetPresignURLSong(ctx context.Context, id string) (string, error) {
	const op = "./internal/services/music.go.GetPresignURL()"

	url, err := s.s3.GetPresignURL(ctx, id)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url, nil
}

func (s *MusicService) GetPresignURCover(ctx context.Context, id string) (string, error) {
	const op = "./internal/services/music.go.GetPresignURLCover()"

	url, err := s.s3.GetPresignURL(ctx, id+songImagePostfix)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return url, nil
}

func (s *MusicService) GetMusicSQ(ctx context.Context, m models.MusicFilterQuery, userID string) ([]models.Music, []models.Like, error) {
	const op = "./internal/services/music.go.GetMusicSQ()"

	mu, l, err := s.repo.GetMusicSQ(ctx, m, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", op, err)
	}

	return mu, l, nil
}

func (s *MusicService) GetMusicByUploaderID(ctx context.Context, id string) ([]models.Music, error) {
	const op = "./internal/services/music.go.GetMusicByUploaderID"

	m, err := s.repo.GetMusicByUploaderID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return m, nil
}

func (s *MusicService) AddListening(ctx context.Context, id string) (error) {
	const op = "./internal/services/music.go.AddListening()"

	err := s.repo.AddListening(ctx, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *MusicService) ListObjects(ctx context.Context, prefix string, suf string) ([]string, error) {
	const op = "./internal/services/music.goListObjects"

	sl, err := s.s3.ListObjects(ctx, prefix, suf)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return sl, nil
}

func (s *MusicService) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	const op = "./internal/services/music.go.GetObject()"

	slog.Info("key", key)

	o, err := s.s3.GetObject(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return o, nil
}