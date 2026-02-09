package main

import (
	"context"
	"dpm/internal/adapters/repo/postgres"
	"dpm/internal/config"
	"fmt"
)

func main() {
	cfg := config.MuslLoad()

	var pgCfg postgres.PgConfig
	pgCfg = postgres.NewPgCfg(cfg.User, cfg.Host, int32(cfg.Port), cfg.Password, cfg.DBname)

	fmt.Println(pgCfg)

	pg, err := postgres.New(context.Background(), pgCfg, []postgres.WithFunc{postgres.WithMinConns(&pgCfg, 3), postgres.WithMaxConns(&pgCfg, 10)})
	if err != nil {
		panic(err)
	}
	_ = pg
}
