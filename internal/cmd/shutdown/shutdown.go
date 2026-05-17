package shutdown

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ShutdownFunc func(ctx context.Context) error

func Graceful(timeout time.Duration, funcs ...ShutdownFunc) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	s := <-sig
	slog.Info("received signal, shutting down", "signal", s)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for _, fn := range funcs {
		if err := fn(ctx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	}
}
