package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"todo.mirumo.net/internal/store"
)

// prune clears out what is done. Like rm, it puts tasks aside rather than
// destroying them, so a prune is something you can take back.
func TestPruneSoftRemovesDoneTasks(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "buy milk"})
	app.Run([]string{"add", "fix the parser"})
	app.Run([]string{"done", "2"})
	out.Reset()

	if code := app.Run([]string{"prune"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "deleted #2: fix the parser") {
		t.Errorf("it should name what went: %q", out.String())
	}

	gone, err := app.Store.Get(2)
	if err != nil {
		t.Fatalf("the row should still be there: %v", err)
	}
	if !gone.Deleted() {
		t.Error("and marked deleted")
	}
	kept, err := app.Store.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if kept.Deleted() {
		t.Error("an open task is not something prune touches")
	}
}

func TestPruneForceDestroysTheRows(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "buy milk"})
	app.Run([]string{"done", "1"})
	out.Reset()

	if code := app.Run([]string{"prune", "-f"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "destroyed #1") {
		t.Errorf("it should say the row is gone, not put aside: %q", out.String())
	}
	if _, err := app.Store.Get(1); err == nil {
		t.Error("-f should leave nothing behind")
	}
}

// Nothing done means nothing to do, and saying so beats silence.
func TestPruneWithNothingToTakeSaysSo(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "buy milk"})
	out.Reset()

	if code := app.Run([]string{"prune"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "nothing to prune") {
		t.Errorf("out = %q", out.String())
	}
}

// -b writes what was taken to a real task database: the file opens with --db,
// and the ids in it are the ids the tasks had.
func TestPruneBackupWritesADatabaseOfWhatWentAway(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "buy milk", "-t", "shopping"})
	app.Run([]string{"add", "still open"})
	app.Run([]string{"done", "1"})
	out.Reset()

	path := filepath.Join(t.TempDir(), "done.sqlite")
	if code := app.Run([]string{"prune", "-f", "-b", path}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), path) {
		t.Errorf("it should say where the backup went: %q", out.String())
	}

	back, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("the backup should open as a database: %v", err)
	}
	defer back.Close()
	got, err := back.Get(1)
	if err != nil {
		t.Fatalf("#1 should be in the backup under its own id: %v", err)
	}
	if got.Title != "buy milk" {
		t.Errorf("title = %q", got.Title)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "shopping" {
		t.Errorf("tags should come along: %v", got.Tags)
	}
	if _, err := back.Get(2); err == nil {
		t.Error("a task prune did not take has no business in the backup")
	}
}

// -b with no value names the file after the time and puts it beside the
// database the tasks came from, wherever --db put that.
func TestPruneBackupWithNoValueLandsBesideTheDatabase(t *testing.T) {
	app, out, errBuf := newApp(t)
	dir := t.TempDir()
	app.DBPath = filepath.Join(dir, "todo.db")
	app.Run([]string{"add", "buy milk"})
	app.Run([]string{"done", "1"})
	out.Reset()

	if code := app.Run([]string{"prune", "-b"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	want := filepath.Join(dir, "20260829T150000_db.sqlite")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("expected a backup at %s: %v", want, err)
	}
	if !strings.Contains(out.String(), want) {
		t.Errorf("it should say where it went: %q", out.String())
	}
}

// The backup comes first, so a backup that cannot be written leaves the list
// exactly as it was.
func TestPruneKeepsEverythingWhenTheBackupFails(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "buy milk"})
	app.Run([]string{"done", "1"})
	out.Reset()

	taken := filepath.Join(t.TempDir(), "taken.sqlite")
	if err := os.WriteFile(taken, []byte("in the way"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := app.Run([]string{"prune", "-f", "-b", taken}); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "backup") {
		t.Errorf("the error should say the backup is what failed: %q", errBuf.String())
	}
	if _, err := app.Store.Get(1); err != nil {
		t.Error("nothing should have been removed")
	}
}

// prune takes ls's filters and ls's defaults with them: the plain command is
// about uncategorized tasks, and reaching a project's needs a flag.
func TestPruneHonoursTheFilters(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "loose end"})
	app.Run([]string{"add", "project work", "-p", "/work"})
	app.Run([]string{"add", "a chore", "-t", "chore"})
	app.Run([]string{"done", "1"})
	app.Run([]string{"done", "2"})
	app.Run([]string{"done", "3"})
	out.Reset()

	if code := app.Run([]string{"prune"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	s := out.String()
	if !strings.Contains(s, "#1") || !strings.Contains(s, "#3") {
		t.Errorf("uncategorized done tasks should go: %q", s)
	}
	if strings.Contains(s, "#2") {
		t.Errorf("a project's task should need -p or --all-projects: %q", s)
	}

	out.Reset()
	if code := app.Run([]string{"prune", "--all-projects"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "#2") {
		t.Errorf("--all-projects should reach it: %q", out.String())
	}
}

func TestPruneByTag(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "a chore", "-t", "chore"})
	app.Run([]string{"add", "something else"})
	app.Run([]string{"done", "1"})
	app.Run([]string{"done", "2"})
	out.Reset()

	if code := app.Run([]string{"prune", "-t", "chore"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "a chore") || strings.Contains(out.String(), "something else") {
		t.Errorf("-t should narrow it to the tag: %q", out.String())
	}
}

// The flags prune has no answer for are not quietly accepted.
func TestPruneRejectsTheFlagsItHasNoUseFor(t *testing.T) {
	app, _, errBuf := newApp(t)
	for _, flag := range []string{"-a", "--done", "-s", "-r"} {
		errBuf.Reset()
		if code := app.Run([]string{"prune", flag}); code == 0 {
			t.Errorf("prune %s should not be accepted", flag)
		}
		// Asserted on the message, not just the code: "--sort needs a value"
		// is also a non-zero exit, and it is not the answer wanted here.
		if !strings.Contains(errBuf.String(), "unknown flag "+flag) {
			t.Errorf("prune %s: err = %q, want it called an unknown flag", flag, errBuf.String())
		}
	}
}

// A done task that rm already put aside is out of prune's way by default, and
// --deleted is how you go in after it.
func TestPruneDeletedEmptiesTheBinOfDoneTasks(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.Run([]string{"add", "buy milk"})
	app.Run([]string{"done", "1"})
	app.Run([]string{"rm", "1"})
	out.Reset()

	if code := app.Run([]string{"prune"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "nothing to prune") {
		t.Errorf("what rm already took should not be taken twice: %q", out.String())
	}

	out.Reset()
	if code := app.Run([]string{"prune", "--deleted", "-f"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "destroyed #1") {
		t.Errorf("--deleted -f should reach it: %q", out.String())
	}
	if _, err := app.Store.Get(1); err == nil {
		t.Error("and destroy it")
	}
}
