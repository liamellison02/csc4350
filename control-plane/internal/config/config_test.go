package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// empty values must fall back to the documented defaults.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OPAMP_LISTEN", "")
	t.Setenv("RECONCILE_INTERVAL", "")

	cfg := Load()

	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
	if cfg.OpAMPListen != defaultOpAMPListen {
		t.Errorf("OpAMPListen = %q, want %q", cfg.OpAMPListen, defaultOpAMPListen)
	}
	if cfg.ReconcileInterval != defaultReconcileInterval {
		t.Errorf("ReconcileInterval = %s, want %s", cfg.ReconcileInterval, defaultReconcileInterval)
	}
}

func TestLoadOverrides(t *testing.T) {
	const (
		wantDB       = "postgres://u:p@db:6543/helm"
		wantListen   = "127.0.0.1:9999"
		wantInterval = 2 * time.Second
	)
	t.Setenv("DATABASE_URL", wantDB)
	t.Setenv("OPAMP_LISTEN", wantListen)
	t.Setenv("RECONCILE_INTERVAL", "2s")

	cfg := Load()

	if cfg.DatabaseURL != wantDB {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, wantDB)
	}
	if cfg.OpAMPListen != wantListen {
		t.Errorf("OpAMPListen = %q, want %q", cfg.OpAMPListen, wantListen)
	}
	if cfg.ReconcileInterval != wantInterval {
		t.Errorf("ReconcileInterval = %s, want %s", cfg.ReconcileInterval, wantInterval)
	}
}

func TestLoadReconcileIntervalInvalid(t *testing.T) {
	// unparseable and nonpositive values silently fall back to the default.
	for _, v := range []string{"soon", "-5s", "0"} {
		t.Setenv("RECONCILE_INTERVAL", v)

		cfg := Load()

		if cfg.ReconcileInterval != defaultReconcileInterval {
			t.Errorf("RECONCILE_INTERVAL=%q: ReconcileInterval = %s, want %s", v, cfg.ReconcileInterval, defaultReconcileInterval)
		}
	}
}
