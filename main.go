package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jozef/clickhouse-alerting-system/internal/api"
	"github.com/jozef/clickhouse-alerting-system/internal/connregistry"
	"github.com/jozef/clickhouse-alerting-system/internal/evaluator"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
	"github.com/jozef/clickhouse-alerting-system/internal/notifier"
	"github.com/jozef/clickhouse-alerting-system/internal/store"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed ui/dist
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

	// Connection registry
	registry := connregistry.New(sqliteStore)
	defer registry.Close()

	// Seed default connection from config if no connections exist yet
	seedDefaultConnection(sqliteStore, cfg)

	// Seed default rules for any connections that have no rules
	seedDefaultRulesForExisting(sqliteStore)

	// Notifier
	dispatcher := notifier.NewDispatcher(sqliteStore)

	// Evaluator
	eval := evaluator.New(
		sqliteStore,
		registry,
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
	srv := api.NewServer(sqliteStore, dispatcher, registry)
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

func seedDefaultConnection(st *store.SQLiteStore, cfg Config) {
	if cfg.ClickHouse.DSN == "" || cfg.ClickHouse.DSN == "clickhouse://default:@localhost:9000/default" {
		return
	}
	ctx := context.Background()
	conns, err := st.ListConnections(ctx)
	if err != nil || len(conns) > 0 {
		return
	}

	// Parse DSN to extract structured fields
	parsed, err := url.Parse(cfg.ClickHouse.DSN)
	if err != nil {
		slog.Warn("could not parse clickhouse DSN for seeding", "error", err)
		return
	}

	host := parsed.Hostname()
	port := 9000
	if p := parsed.Port(); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}
	database := strings.TrimPrefix(parsed.Path, "/")
	if database == "" {
		database = "default"
	}
	username := "default"
	password := ""
	if parsed.User != nil {
		username = parsed.User.Username()
		password, _ = parsed.User.Password()
	}
	secure := parsed.Query().Get("secure") == "true"

	maxConns := cfg.ClickHouse.MaxOpenConns
	if maxConns <= 0 {
		maxConns = 5
	}

	now := time.Now().UTC()
	conn := model.ClickHouseConnection{
		ID:           uuid.New().String(),
		Name:         "Default",
		Host:         host,
		Port:         port,
		Database:     database,
		Username:     username,
		Password:     password,
		Secure:       secure,
		MaxOpenConns: maxConns,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := st.CreateConnection(ctx, conn); err != nil {
		slog.Warn("failed to seed default connection", "error", err)
		return
	}
	slog.Info("seeded default clickhouse connection from config", "name", conn.Name, "host", conn.Host)
}

func seedDefaultRulesForExisting(st *store.SQLiteStore) {
	ctx := context.Background()
	conns, err := st.ListConnections(ctx)
	if err != nil || len(conns) == 0 {
		return
	}
	for _, conn := range conns {
		rules, err := st.ListRulesByConnection(ctx, conn.ID)
		if err != nil {
			continue
		}
		if len(rules) == 0 {
			api.SeedDefaultRules(ctx, st, conn.ID)
		}
	}
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
