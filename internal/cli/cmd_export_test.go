package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedForExport leaves one of everything: open, done, deleted, and one in a
// project, so a dump's defaults can be seen.
func seedForExport(t *testing.T, app *App) {
	t.Helper()
	app.Run([]string{"add", "open one", "-t", "shopping", "--desc", "two\nlines"})
	app.Run([]string{"add", "done one"})
	app.Run([]string{"add", "deleted one"})
	app.Run([]string{"add", "work one", "-p", "/p/work"})
	app.Run([]string{"done", "2"})
	app.Run([]string{"rm", "3"})
}

// A dump is a backup, so it takes everything unless it is told to narrow: the
// ls defaults would quietly leave out done, deleted and every project.
func TestExportTakesEverythingByDefault(t *testing.T) {
	app, out, _ := newApp(t)
	seedForExport(t, app)
	out.Reset()

	if code := app.Run([]string{"export"}); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("the default format should be json: %v\n%s", err, out.String())
	}
	if len(got) != 4 {
		t.Fatalf("a dump should hold all four tasks, got %d: %s", len(got), out.String())
	}
}

// The ls filters still narrow it.
func TestExportTakesTheLsFilters(t *testing.T) {
	app, out, _ := newApp(t)
	seedForExport(t, app)
	out.Reset()

	app.Run([]string{"export", "-t", "shopping"})
	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["title"] != "open one" {
		t.Errorf("= %v", got)
	}
}

func TestDumpIsTheSameCommand(t *testing.T) {
	app, out, _ := newApp(t)
	seedForExport(t, app)
	out.Reset()
	app.Run([]string{"dump", "-f", "yaml"})
	if !strings.Contains(out.String(), `title: "open one"`) {
		t.Errorf("dump should behave like export:\n%s", out.String())
	}
}

func TestExportToAFile(t *testing.T) {
	app, out, errBuf := newApp(t)
	seedForExport(t, app)
	out.Reset()
	path := filepath.Join(t.TempDir(), "tasks.yaml")

	if code := app.Run([]string{"export", "-o", path}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The extension chose the format, with no -f needed.
	if !strings.Contains(string(b), `title: "open one"`) {
		t.Errorf("the file should be yaml:\n%s", b)
	}
	// Writing to a file leaves the terminal free to say what happened.
	if !strings.Contains(out.String(), "4 tasks") {
		t.Errorf("it should report what it wrote: %q", out.String())
	}
}

// Nothing but the dump may reach stdout, or the dump is not a dump.
func TestExportToStdoutSaysNothingElse(t *testing.T) {
	app, out, _ := newApp(t)
	seedForExport(t, app)
	out.Reset()
	app.Run([]string{"export", "-f", "json"})
	if !strings.HasPrefix(strings.TrimSpace(out.String()), "[") {
		t.Errorf("stdout should be the dump and nothing else:\n%s", out.String())
	}
}

func TestExportFormatFlagWinsOverTheExtension(t *testing.T) {
	app, _, errBuf := newApp(t)
	seedForExport(t, app)
	path := filepath.Join(t.TempDir(), "tasks.yaml")

	if code := app.Run([]string{"export", "-o", path, "-f", "json"}); code != 0 {
		t.Fatalf("exit code = %d: %s", code, errBuf.String())
	}
	b, _ := os.ReadFile(path)
	if !strings.HasPrefix(strings.TrimSpace(string(b)), "[") {
		t.Errorf("-f should win over the extension:\n%s", b)
	}
}

func TestExportRejectsAnUnknownFormat(t *testing.T) {
	app, _, errBuf := newApp(t)
	if code := app.Run([]string{"export", "-f", "xml"}); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "xml") {
		t.Errorf("the error should name it: %q", errBuf.String())
	}
}

func TestExportReportsAnUnwritableFile(t *testing.T) {
	app, _, errBuf := newApp(t)
	if code := app.Run([]string{"export", "-o", filepath.Join(t.TempDir(), "no", "such", "dir", "x.json")}); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if errBuf.Len() == 0 {
		t.Error("it should say what went wrong")
	}
}
