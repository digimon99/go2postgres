package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolManager maintains lazily-initialised per-instance connection pools.
// It is safe for concurrent use.
type PoolManager struct {
	mu    sync.RWMutex
	pools map[string]*pgxpool.Pool // key: instanceID
}

// NewPoolManager creates a new PoolManager.
func NewPoolManager() *PoolManager {
	return &PoolManager{pools: make(map[string]*pgxpool.Pool)}
}

// GetPool returns (or lazily creates) a pool for instanceID.
// dsn is the full postgres connection string.
// maxConns limits the pool size (≤0 defaults to 5).
// stmtTimeoutMs is passed as default_query_exec_timeout (pgx v5) via a pool config.
func (pm *PoolManager) GetPool(ctx context.Context, instanceID, dsn string, maxConns int32, stmtTimeoutMs int) (*pgxpool.Pool, error) {
	// Fast path
	pm.mu.RLock()
	if p, ok := pm.pools[instanceID]; ok {
		pm.mu.RUnlock()
		return p, nil
	}
	pm.mu.RUnlock()

	// Slow path — create pool under write lock
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Double-check after acquiring write lock
	if p, ok := pm.pools[instanceID]; ok {
		return p, nil
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing dsn for instance %s: %w", instanceID, err)
	}

	if maxConns <= 0 {
		maxConns = 5
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 15 * time.Minute
	cfg.MaxConnIdleTime = 3 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	if stmtTimeoutMs > 0 {
		// Apply per-connection statement timeout via pgx connect hook.
		cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
			_, err := c.Exec(ctx, fmt.Sprintf("SET statement_timeout = '%dms'", stmtTimeoutMs))
			return err
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool for instance %s: %w", instanceID, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging pool for instance %s: %w", instanceID, err)
	}

	pm.pools[instanceID] = pool
	return pool, nil
}

// Evict closes and removes the pool for a single instance.
func (pm *PoolManager) Evict(instanceID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if p, ok := pm.pools[instanceID]; ok {
		p.Close()
		delete(pm.pools, instanceID)
	}
}

// CloseAll closes all managed pools.
func (pm *PoolManager) CloseAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for id, p := range pm.pools {
		p.Close()
		delete(pm.pools, id)
	}
}
