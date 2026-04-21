package main

import (
	"context"
	"dpm/internal/adapters/http"
	"dpm/internal/adapters/repo/postgres"
	"dpm/internal/config"
	"dpm/internal/models"
	"dpm/internal/services"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

func script(ms *services.MusicService) {
	rand.NewSource(time.Now().Unix())
	for range 5 {
		err := ms.CreateMusic(context.Background(), models.Music{
			Name:        "SomeMusic",
			UploaderID:  "75e14016-ba7f-45bd-835b-b13dcac46de7",
			Likes:       rand.Intn(101) + 1,
			DurationSec: rand.Intn(101) + 60,
		})
		if err != nil {
			slog.Error(err.Error())
		}
	}
}

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

	uService := services.NewUser(pg, cfg.JWT.Key)

	err = uService.RegisterUser(context.Background(), u)

	// err = uService.RegisterUser(context.Background(), u)
	// if err != nil {
	// 	slog.Error(err.Error())
	// }

	// token, err := uService.Login(context.Background(), u)
	// if err != nil {
	// 	slog.Error(err.Error())
	// }
	// slog.Info(token)

	mService := services.NewMusicService(pg)

	lhService := services.NewListeningHistoryService(pg)

	fService := services.NewFavorService(pg)

	likeService := services.NewLikeService(pg)

	handler := http.NewHandler(uService, mService, lhService, fService, likeService)

	server := http.NewServer(handler)

	slog.Info(server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()

	script(mService)

	slog.Info("server start")

	err = <-errChan
	slog.Error(err.Error())
}
