// Package db provides the SQLite data layer for ntts-flightlog v2.
// schema.go defines Go-side constants for table/view names and domain values.
package db

// Table names.
const (
	TableSessions              = "sessions"
	TableTurns                 = "turns"
	TableEntries               = "entries"
	TableBlockers              = "blockers"
	TableDecisionEvidenceLinks = "decision_evidence_links"
	TableDecisionStatus        = "decision_status"
	TableSchemaMigrations      = "_schema_migrations"
)

// View names (filtered over entries by kind).
const (
	ViewDecisions = "decisions"
	ViewEvidence  = "evidence"
)

// Entry kind values — matches CHECK constraint in 0001_init.sql.
const (
	KindEntry    = "entry"
	KindDecision = "decision"
	KindEvidence = "evidence"
	KindBlocker  = "blocker"
	KindMode     = "mode"
)

// Turn status values.
const (
	TurnStatusActive   = "active"
	TurnStatusComplete = "complete"
	TurnStatusAbort    = "abort"
	TurnStatusAbandon  = "abandon"
)

// Blocker status values.
const (
	BlockerStatusOpen     = "open"
	BlockerStatusResolved = "resolved"
)

// Decision status values.
const (
	DecisionStatusAccepted   = "accepted"
	DecisionStatusSuperseded = "superseded"
	DecisionStatusRejected   = "rejected"
)

// Session mode values (non-exhaustive; user can set arbitrary modes).
const (
	ModeSolo = "solo"
	ModePlan = "plan"
	ModeAuto = "auto"
)
