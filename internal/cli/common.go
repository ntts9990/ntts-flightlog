package cli

import (
	"fmt"
	"os"
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
	lane     string   // --lane flag value (empty is default lane)
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
	lane := worklog.SafeLaneName(laneFlag)
	if err := cfg.EnsureLaneDir(lane); err != nil {
		_ = store.Close()
		return nil, err
	}

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
		lane:     lane,
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

func (s *session) activeTurnID() string {
	return s.cfg.ActiveTurnIDForLane(s.lane)
}

func (s *session) activeTurnNumber() int {
	return s.cfg.ActiveTurnNumberForLane(s.lane)
}

func (s *session) writeActiveTurn(turnID string, turnNumber int) error {
	if err := s.cfg.EnsureLaneDir(s.lane); err != nil {
		return err
	}
	if err := worklog.WriteFile(s.cfg.LaneTurnIDFile(s.lane), turnID); err != nil {
		return err
	}
	if err := worklog.WriteFile(s.cfg.LaneTurnNumberFile(s.lane), fmt.Sprintf("%d", turnNumber)); err != nil {
		return err
	}
	return worklog.WriteFile(s.cfg.LaneTurnStartFile(s.lane), worklog.EpochSeconds())
}

func (s *session) clearActiveTurn() {
	_ = os.Remove(s.cfg.LaneTurnStartFile(s.lane))
	_ = os.Remove(s.cfg.LaneTurnIDFile(s.lane))
	if s.lane != "" {
		_ = os.Remove(s.cfg.LaneTurnNumberFile(s.lane))
	}
}
