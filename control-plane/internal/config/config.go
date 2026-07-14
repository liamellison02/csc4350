// Package config loads control plane settings from the environment.
package config

import (
	"os"
	"time"
)

// default connection string, listen endpoint, and reconcile cadence.
const (
	defaultDatabaseURL       = "postgres://helmsman:helmsman@localhost:5432/helmsman"
	defaultOpAMPListen       = ":4320"
	defaultReconcileInterval = 10 * time.Second
)

// Config holds runtime settings sourced from the environment.
type Config struct {
	DatabaseURL       string
	OpAMPListen       string
	ReconcileInterval time.Duration
}

// Load reads config from the environment, applying defaults for any
// unset or empty variable.
func Load() Config {
	interval := defaultReconcileInterval
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			interval = d
		}
	}
	return Config{
		DatabaseURL:       getenv("DATABASE_URL", defaultDatabaseURL),
		OpAMPListen:       getenv("OPAMP_LISTEN", defaultOpAMPListen),
		ReconcileInterval: interval,
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
