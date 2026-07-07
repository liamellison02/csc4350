package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	// empty values must fall back to the documented defaults.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OPAMP_LISTEN", "")

	cfg := Load()

	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
	if cfg.OpAMPListen != defaultOpAMPListen {
		t.Errorf("OpAMPListen = %q, want %q", cfg.OpAMPListen, defaultOpAMPListen)
	}
}

func TestLoadOverrides(t *testing.T) {
	const (
		wantDB     = "postgres://u:p@db:6543/helm"
		wantListen = "127.0.0.1:9999"
	)
	t.Setenv("DATABASE_URL", wantDB)
	t.Setenv("OPAMP_LISTEN", wantListen)

	cfg := Load()

	if cfg.DatabaseURL != wantDB {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, wantDB)
	}
	if cfg.OpAMPListen != wantListen {
		t.Errorf("OpAMPListen = %q, want %q", cfg.OpAMPListen, wantListen)
	}
}
