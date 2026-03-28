package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jacinli/sky-guardwall/internal/config"
	"github.com/jacinli/sky-guardwall/internal/database"
	"github.com/jacinli/sky-guardwall/internal/frontend"
	"github.com/jacinli/sky-guardwall/internal/router"
	"github.com/jacinli/sky-guardwall/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()
	db := database.Init(cfg)
	iptablesSvc := service.NewIptablesService(db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Background iptables sync every cfg.SyncIntervalSecs (default 60s)
	go func() {
		slog.Info("running initial iptables sync")
		iptablesSvc.Sync(context.Background())

		ticker := time.NewTicker(time.Duration(cfg.SyncIntervalSecs) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				iptablesSvc.Sync(context.Background())
			case <-ctx.Done():
				slog.Info("sync goroutine stopped")
				return
			}
		}
	}()

	r := router.Setup(cfg, db, frontend.FS)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		slog.Info("SkyGuardwall started", "addr", "http://0.0.0.0:"+cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	slog.Info("server stopped")
}
