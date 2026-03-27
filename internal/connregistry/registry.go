package connregistry

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jozef/clickhouse-alerting-system/internal/model"
	"github.com/jozef/clickhouse-alerting-system/internal/store"
)

type Registry struct {
	mu    sync.RWMutex
	conns map[string]*sql.DB
	store store.Store
}

func New(st store.Store) *Registry {
	return &Registry{
		conns: make(map[string]*sql.DB),
		store: st,
	}
}

// Get returns the *sql.DB for a connection ID, opening it lazily if needed.
func (r *Registry) Get(ctx context.Context, id string) (*sql.DB, error) {
	r.mu.RLock()
	db, ok := r.conns[id]
	r.mu.RUnlock()
	if ok {
		return db, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if db, ok := r.conns[id]; ok {
		return db, nil
	}

	conn, err := r.store.GetConnection(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("loading connection config: %w", err)
	}
	if !conn.Enabled {
		return nil, fmt.Errorf("connection %q is disabled", conn.Name)
	}

	db, err = sql.Open("clickhouse", conn.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening connection %q: %w", conn.Name, err)
	}
	db.SetMaxOpenConns(conn.MaxOpenConns)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging connection %q: %w", conn.Name, err)
	}

	r.conns[id] = db
	slog.Info("opened clickhouse connection", "name", conn.Name, "host", conn.Host)
	return db, nil
}

// Invalidate closes and removes a cached connection. Called on create/update/delete.
func (r *Registry) Invalidate(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if db, ok := r.conns[id]; ok {
		db.Close()
		delete(r.conns, id)
	}
}

// Close closes all cached connections.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, db := range r.conns {
		db.Close()
		delete(r.conns, id)
	}
}

// TestConnection opens a temporary connection, pings it, and closes it.
func TestConnection(ctx context.Context, conn model.ClickHouseConnection) error {
	db, err := sql.Open("clickhouse", conn.DSN())
	if err != nil {
		return fmt.Errorf("opening connection: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}
