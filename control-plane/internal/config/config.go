// Package config loads control plane settings from the environment.
package config

import "os"

// default connection string and listen endpoint.
const (
	defaultDatabaseURL = "postgres://helmsman:helmsman@localhost:5432/helmsman"
	defaultOpAMPListen = ":4320"
)

// Config holds runtime settings sourced from the environment.
type Config struct {
	DatabaseURL string
	OpAMPListen string
}

// Load reads config from the environment, applying defaults for any
// unset or empty variable.
func Load() Config {
	return Config{
		DatabaseURL: getenv("DATABASE_URL", defaultDatabaseURL),
		OpAMPListen: getenv("OPAMP_LISTEN", defaultOpAMPListen),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
