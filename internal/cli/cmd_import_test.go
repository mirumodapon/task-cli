package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"todo.mirumo.net/internal/task"
)

// dumped exports everything the app holds, as the given format.
func dumped(t *testing.T, app *App, out *strings.Builder, format string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tasks."+format)
	if code := app.Run([]string{"export", "-o", path}); code != 0 {
		t.Fatalf("export: exit code = %d", code)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// What export writes, import reads: for both the formats meant for machines.
func TestImportRoundTrip(t *testing.T) {
	for _, format := range []string{"json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			from, _, _ := newApp(t)
			from.Run([]string{"add", "buy milk", "-t", "shopping", "-d", "2026-09-12", "--pri", "high", "--desc", "two\nlines"})
			from.Run([]string{"add", "work one", "-p", "/p/work"})
			var sb strings.Builder
			text := dumped(t, from, &sb, format)

			to, out, errBuf := newApp(t)
			path := filepath.Join(t.TempDir(), "in."+format)
			if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
				t.Fatal(err)
			}
			if code := to.Run([]string{"import", path}); code != 0 {
				t.Fatalf("import: exit code = %d: %s", code, errBuf.String())
			}
			if !strings.Contains(out.String(), "2 tasks") {
				t.Errorf("it should say what it took in: %q", out.String())
			}

			got, err := to.Store.Get(1)
			if err != nil {
				t.Fatal(err)
			}
			if got.Title != "buy milk" || got.Desc != "two\nlines" {
				t.Errorf("= %+v", got)
			}
			if len(got.Tags) != 1 || got.Tags[0] != "shopping" {
				t.Errorf("tags = %v", got.Tags)
			}
			if got.Due == nil || got.Due.Format("2006-01-02") != "2026-09-12" {
				t.Errorf("due = %v", got.Due)
			}
			second, _ := to.Store.Get(2)
			if second.Project != "/p/work" {
				t.Errorf("project = %q", second.Project)
			}
		})
	}
}

// No file means stdin, so a dump can be piped straight in.
func TestImportFromStdin(t *testing.T) {
	app, out, errBuf := newApp(t)
	app.In = strings.NewReader(`[{"id":9,"title":"piped in","created_at":"2026-08-29T15:00:00+08:00","updated_at":"2026-08-29T15:00:00+08:00"}]`)
	if code := app.Run([]string{"import"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "1 task") {
		t.Errorf("stdout = %q", out.String())
	}
	got, err := app.Store.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "piped in" {
		t.Errorf("= %+v", got)
	}
}

// Ids are the importing list's to give: the ones in the file belong to the list
// it came from, and two lists have no reason to agree.
func TestImportGivesTasksNewIDs(t *testing.T) {
	app, _, _ := newApp(t)
	app.Run([]string{"add", "already here"})
	app.In = strings.NewReader(`[{"id":1,"title":"from the file","created_at":"2026-08-29T15:00:00+08:00","updated_at":"2026-08-29T15:00:00+08:00"}]`)
	if code := app.Run([]string{"import"}); code != 0 {
		t.Fatal("import should not collide with what is already there")
	}
	first, _ := app.Store.Get(1)
	second, _ := app.Store.Get(2)
	if first.Title != "already here" || second.Title != "from the file" {
		t.Errorf("= %q, %q", first.Title, second.Title)
	}
}

// A file that is not a dump changes nothing: parsing finishes before anything
// is written.
func TestImportOfSomethingElseChangesNothing(t *testing.T) {
	app, _, errBuf := newApp(t)
	app.In = strings.NewReader("this is not a dump\n")
	if code := app.Run([]string{"import"}); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if errBuf.Len() == 0 {
		t.Error("it should say why")
	}
	ts, _ := app.Store.List(task.Filter{IncludeDone: true, IncludeDeleted: true}, refTime())
	if len(ts) != 0 {
		t.Errorf("nothing should have been written, got %d tasks", len(ts))
	}
}

// txt and sql say what to do instead rather than half working.
func TestImportRefusesTheFormatsThatCannotComeBack(t *testing.T) {
	for _, format := range []string{"txt", "sql"} {
		app, _, errBuf := newApp(t)
		app.In = strings.NewReader("anything")
		if code := app.Run([]string{"import", "-f", format}); code != 1 {
			t.Errorf("%s: exit code = %d, want 1", format, code)
		}
		if errBuf.Len() == 0 {
			t.Errorf("%s: it should say why not", format)
		}
	}
}

func TestImportReportsAMissingFile(t *testing.T) {
	app, _, errBuf := newApp(t)
	if code := app.Run([]string{"import", filepath.Join(t.TempDir(), "nope.json")}); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "nope.json") {
		t.Errorf("the error should name the file: %q", errBuf.String())
	}
}

func TestImportTakesAtMostOneFile(t *testing.T) {
	app, _, _ := newApp(t)
	if code := app.Run([]string{"import", "a.json", "b.json"}); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}
