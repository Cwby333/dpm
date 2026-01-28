package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WithFunc func(PgConfig)

type Postgres struct {	
	pool *pgxpool.Pool
}

type PgConfig struct {
	user string
	host string
	port int32
	password string
	dbName string
	minConns int
	maxConns int
}

func WithMinConns(pgcfg *PgConfig, minConns int) WithFunc {
	return func(pc PgConfig) {
		pgcfg.minConns = minConns
	}
}
func WithMaxConns(pgcfg *PgConfig, maxConns int) WithFunc {
	return func(pc PgConfig) {
		pgcfg.maxConns = maxConns
	}
}

func NewPgCfg(user string, host string, port int32, password string, dbName string) PgConfig {	
	pgcfg := PgConfig{
		user: user,
		host: host,
		port: port,
		password: password,
		dbName: dbName,
	}

	if pgcfg.minConns == 0 {
		pgcfg.minConns = 5
	}
	if pgcfg.maxConns == 0 {
		pgcfg.maxConns = 20
	}

	return pgcfg
}

func New(ctx context.Context, cfg PgConfig,  funcs[] WithFunc) (Postgres, error) {
	const op = "./internal/adapters/postgres/init.go"

	for i := range funcs {
		funcs[i](cfg)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&pool_max_conns=%d&pool_min_conns=%d",
		cfg.user,
		cfg.password,
		cfg.host,
		cfg.port,
		cfg.dbName,
		cfg.maxConns,
		cfg.minConns,
	)

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return Postgres{}, fmt.Errorf("%s: %w", op, err)
	}

	pgxpool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return Postgres{}, fmt.Errorf("%s: %w", op, err)
	}

	err = pgxpool.Ping(ctx)
	if err != nil {
		return Postgres{}, fmt.Errorf("%s: %w", op, err)
	}

	slog.Info("success connect postgres")

	return Postgres{
		pool: pgxpool,
	}, nil
}