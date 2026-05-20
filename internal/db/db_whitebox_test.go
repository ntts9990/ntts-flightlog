package db

// db_whitebox_test.go: white-box tests (package db) that call unexported
// functions with injected failures to cover error-return branches.

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// TestApplyPRAGMAsClosedDB verifies applyPRAGMAs returns an error when the
// underlying sql.DB is already closed, covering the error-return branch.
func TestApplyPRAGMAsClosedDB(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	// Force a live connection before closing (ensures the connection pool
	// is initialised and the Close is meaningful).
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	_ = sqlDB.Close()

	err = applyPRAGMAs(sqlDB)
	if err == nil {
		t.Error("applyPRAGMAs on closed DB: expected error, got nil")
	}
}

// TestMigrateOnClosedDB verifies migrate() returns an error when the
// underlying sql.DB is closed, covering the createMeta error-return branch.
func TestMigrateOnClosedDB(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	_ = sqlDB.Close()

	d := &DB{sqlDB}
	err = d.migrate()
	if err == nil {
		t.Error("migrate on closed DB: expected error, got nil")
	}
}
