package main

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jozef/clickhouse-alerting-system/internal/api"
	"github.com/jozef/clickhouse-alerting-system/internal/evaluator"
	"github.com/jozef/clickhouse-alerting-system/internal/notifier"
	"github.com/jozef/clickhouse-alerting-system/internal/store"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed ui/index.html ui/app.js ui/style.css
var uiFS embed.FS

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	setupLogging(cfg.Log)

	// Set embedded filesystems
	store.MigrationsFS = migrationsFS
	api.UIFS = uiFS

	// Open SQLite
	sqliteStore, err := store.NewSQLiteStore(cfg.SQLite.Path)
	if err != nil {
		slog.Error("failed to open sqlite", "error", err)
		os.Exit(1)
	}
	defer sqliteStore.Close()
	slog.Info("sqlite initialized", "path", cfg.SQLite.Path)

	// Open ClickHouse
	chDB, err := sql.Open("clickhouse", cfg.ClickHouse.DSN)
	if err != nil {
		slog.Error("failed to open clickhouse", "error", err)
		os.Exit(1)
	}
	defer chDB.Close()
	chDB.SetMaxOpenConns(cfg.ClickHouse.MaxOpenConns)
	slog.Info("clickhouse configured", "dsn", cfg.ClickHouse.DSN)

	// Notifier
	dispatcher := notifier.NewDispatcher(sqliteStore)

	// Evaluator
	eval := evaluator.New(
		sqliteStore,
		chDB,
		cfg.Evaluation.QueryTimeout.Duration,
		cfg.Evaluation.MaxConcurrent,
		cfg.Notification.RepeatInterval.Duration,
		dispatcher.NotifyFunc(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eval.Start(ctx)
	slog.Info("evaluator started")

	// HTTP server
	srv := api.NewServer(sqliteStore, dispatcher)
	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: srv.Handler(),
	}

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("http server starting", "addr", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	slog.Info("shutting down...")

	cancel()
	eval.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)

	slog.Info("shutdown complete")
}

func setupLogging(cfg LogConfig) {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}
	slog.SetDefault(slog.New(handler))
}
