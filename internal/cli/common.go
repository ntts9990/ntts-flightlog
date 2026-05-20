package cli

import (
	"fmt"
	"time"

	"github.com/ntts9990/ntts-flightlog/internal/agent"
	"github.com/ntts9990/ntts-flightlog/internal/db"
	"github.com/ntts9990/ntts-flightlog/internal/worklog"
)

// session groups the runtime context for a single CLI invocation.
type session struct {
	cfg      *worklog.Config
	store    *db.DB
	detected string   // auto-detected agent
	signals  []string // detection evidence
	agentID  string   // effective agent: override ?? detected ?? ""
	override string   // --agent flag value (empty if not set)
}

// openSession opens the worklog config and SQLite DB, runs agent detection,
// and applies the --agent override flag if set.
func openSession() (*session, error) {
	cfg := worklog.DefaultConfig()
	// Ensure the worklog directory exists before opening the DB.
	if err := cfg.EnsureDir(); err != nil {
		return nil, err
	}
	store, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	detected, signals := agent.Detect()
	override := agentFlag // global from root.go

	effective := detected
	if override != "" {
		effective = override
	}
	if effective == "unknown" {
		effective = ""
	}

	return &session{
		cfg:      cfg,
		store:    store,
		detected: detected,
		signals:  signals,
		agentID:  effective,
		override: override,
	}, nil
}

func (s *session) close() {
	_ = s.store.Close()
}

// now returns current UTC time in RFC3339 / ISO 8601.
func now() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// nullStr returns a *string nil when empty (for nullable SQL columns).
func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
