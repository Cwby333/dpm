package main

import (
	"context"
	"dpm/internal/adapters/http"
	objectstorage "dpm/internal/adapters/repo/objectStorage"
	"dpm/internal/adapters/repo/postgres"
	"dpm/internal/config"
	"dpm/internal/models"
	"dpm/internal/services"
	"fmt"
	"log"
	"log/slog"
	// "math/rand"
	// "time"
)

// func script(ms *services.MusicService) {
// 	rand.NewSource(time.Now().Unix())
// 	for range 5 {
// 		err := ms.CreateMusic(context.Background(), models.Music{
// 			Name:        "SomeMusic",
// 			UploaderID:  "75e14016-ba7f-45bd-835b-b13dcac46de7",
// 			Likes:       rand.Intn(101) + 1,
// 			DurationSec: rand.Intn(101) + 60,
// 		})
// 		if err != nil {
// 			slog.Error(err.Error())
// 		}
// 	}
// }

func main() {
	cfg := config.MuslLoad()

	errChan := make(chan error, 1)

	var pgCfg postgres.PgConfig
	pgCfg = postgres.NewPgCfg(cfg.User, cfg.Host, int32(cfg.Port), cfg.Password, cfg.DBname)

	fmt.Println(pgCfg)

	pg, err := postgres.New(context.Background(), pgCfg, []postgres.WithFunc{postgres.WithMinConns(&pgCfg, 3), postgres.WithMaxConns(&pgCfg, 10)})
	if err != nil {
		panic(err)
	}
	_ = pg

	u := models.User{
		Username: "user",
		HashPsw:  "12345678",
		Email:    "email@gmail.com",
	}

	lhService := services.NewListeningHistoryService(pg)

	fService := services.NewFavorService(pg)

	likeService := services.NewLikeService(pg)

	s3, err := objectstorage.NewS3Client(context.Background(), cfg.S3)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %s", "Connect to s3: ", err.Error()))
	}
	_ = s3
	slog.Info("Success connect to s3")

	mService := services.NewMusicService(pg, s3)
	aServices := services.NewAlbumServices(pg, s3)
	pServices := services.NewPlaylistService(pg, s3)

	minl := 44
	maxl := 100
	m, l, err := mService.GetMusicSQ(context.Background(), models.MusicFilterQuery{LikeMin: &minl, LikeMax: &maxl}, "242169af-f0af-47a4-b0f9-f3d7336dab9c")
	if err != nil {
		log.Fatal(err.Error())
	}
	_ = l
	slog.Info(fmt.Sprintf("GetMusicSQResponse: %v", m))
	slog.Info(fmt.Sprintf("GetMusicSQResponseLikes: %v", l))

	uService := services.NewUser(pg, s3, cfg.JWT.Key)

	err = uService.RegisterUser(context.Background(), u)

	handler := http.NewHandler(uService, mService, lhService, fService, likeService, aServices, pServices)

	server := http.NewServer(handler)

	slog.Info(server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()

	// script(mService)

	slog.Info("server start")

	err = <-errChan
	slog.Error(err.Error())
}
