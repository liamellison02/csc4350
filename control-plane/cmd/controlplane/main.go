// Command controlplane runs the helmsman opamp control plane: it accepts
// collector connections, persists agent state to postgres, and marks agents
// disconnected when their connections close.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/liamellison02/csc4350/control-plane/internal/config"
	"github.com/liamellison02/csc4350/control-plane/internal/opamp"
	"github.com/liamellison02/csc4350/control-plane/internal/reconcile"
	"github.com/liamellison02/csc4350/control-plane/internal/store"
)

// connectTimeout bounds the initial postgres reachability check.
const connectTimeout = 10 * time.Second

func main() {
	logger := log.New(os.Stdout, "controlplane ", log.LstdFlags|log.LUTC)
	if err := run(logger); err != nil {
		logger.Fatalf("fatal: %v", err)
	}
}

func run(logger *log.Logger) error {
	cfg := config.Load()

	pool, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	st := store.New(pool)
	srv := opamp.NewServer(st, cfg.OpAMPListen, logger)
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start opamp server on %s: %w", cfg.OpAMPListen, err)
	}
	logger.Printf("opamp control plane listening on %s", cfg.OpAMPListen)

	// push desired configs to drifted agents until shutdown.
	reconcileCtx, stopReconcile := context.WithCancel(context.Background())
	rec := reconcile.New(st, srv, logger)
	go rec.Run(reconcileCtx, cfg.ReconcileInterval)
	logger.Printf("reconciler running every %s", cfg.ReconcileInterval)

	// block until a termination signal, then shut down cleanly.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Printf("received %s, shutting down", sig)

	stopReconcile()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	if err := srv.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("stop opamp server: %w", err)
	}
	logger.Print("shutdown complete")
	return nil
}

// connectDB opens the pool and fails fast with a clear message if postgres
// is unreachable.
func connectDB(url string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("configure postgres pool (%s): %w", url, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres unreachable at %s: %w", url, err)
	}
	return pool, nil
}
