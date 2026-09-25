package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"todo.mirumo.net/internal/task"
)

// Snapshot writes a real task database, not a dump: what comes back out is
// what went in, ids and tags included, so the file can be opened with --db.
func TestSnapshotRoundTripsThroughAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.sqlite")

	done := ref()
	ts := []task.Task{
		{ID: 7, Title: "buy milk", Project: "/home", Due: day(2026, 9, 1), Priority: task.PriHigh,
			Tags: []string{"shopping"}, DoneAt: &done, CreatedAt: ref(), UpdatedAt: ref()},
		{ID: 12, Title: "fix the parser", DoneAt: &done, CreatedAt: ref(), UpdatedAt: ref()},
	}
	if err := Snapshot(path, ts); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	back, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("the snapshot should open as a database: %v", err)
	}
	defer back.Close()

	got, err := back.Get(7)
	if err != nil {
		t.Fatalf("the ids should be the ones it was given: %v", err)
	}
	if got.Title != "buy milk" || got.Project != "/home" || got.Priority != task.PriHigh {
		t.Errorf("fields did not survive: %+v", got)
	}
	if got.DoneAt == nil {
		t.Error("and it should still be done")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "shopping" {
		t.Errorf("tags = %v, want [shopping]", got.Tags)
	}
	if _, err := back.Get(12); err != nil {
		t.Errorf("#12: %v", err)
	}
}

// A backup that silently writes over an earlier backup is worse than one that
// refuses: the file you were counting on is the one that would be gone.
func TestSnapshotRefusesToOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.sqlite")
	if err := os.WriteFile(path, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Snapshot(path, []task.Task{{ID: 1, Title: "x", CreatedAt: ref(), UpdatedAt: ref()}})
	if err == nil {
		t.Fatal("Snapshot should refuse a path that already exists")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the error should name the file: %v", err)
	}
	b, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(b) != "not a database" {
		t.Error("and it should leave what was there alone")
	}
}
