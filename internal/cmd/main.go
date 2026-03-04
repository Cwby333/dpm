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
	// "time"
)

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
		HashPsw: "12345678",
		Email: "email@gmail.com",
	}

	uService := services.NewUser(&pg, cfg.JWT.Key)

	err =  uService.RegisterUser(context.Background(), u)

	// err = uService.RegisterUser(context.Background(), u)
	// if err != nil {
	// 	slog.Error(err.Error())
	// }

	// token, err := uService.Login(context.Background(), u)
	// if err != nil {
	// 	slog.Error(err.Error())
	// }
	// slog.Info(token)

	handler := http.NewHandler(uService)

	server := http.NewServer(handler)

	slog.Info(server.Addr)

	go func ()  {
		if err := server.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()

	slog.Info("server start")

	err = <- errChan
	slog.Error(err.Error())
}
