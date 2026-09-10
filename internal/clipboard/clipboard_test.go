package clipboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTool puts a script named after a real clipboard tool on the PATH, so the
// test can watch what would have been copied without a clipboard to copy to.
func fakeTool(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return dir
}

func TestCopyFeedsTheToolOnStdin(t *testing.T) {
	// An absolute path: PATH has been replaced with the temp dir, so the script
	// cannot look anything up either.
	dir := fakeTool(t, "pbcopy", `/bin/cat > "$0.out"`)
	if err := Copy("two\nlines"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "pbcopy.out"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "two\nlines" {
		t.Errorf("the tool was given %q", got)
	}
}

// The tools are tried in order, so a machine with several gets a predictable one.
func TestCopyPrefersTheFirstToolItFinds(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"xclip", "wl-copy"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\necho "+name+" > \"$0.out\"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	if err := Copy("x"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "wl-copy.out")); err != nil {
		t.Errorf("wl-copy should have been chosen over xclip: %v", err)
	}
}

// A machine with none of them says which ones it looked for, rather than
// failing in a way that reads like the key did nothing.
func TestCopyWithoutAnyToolSaysWhatItLookedFor(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := Copy("x")
	if err == nil {
		t.Fatal("it should be an error")
	}
	for _, want := range []string{"pbcopy", "wl-copy", "xclip"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message should name %q: %v", want, err)
		}
	}
}

func TestCopyReportsAFailingTool(t *testing.T) {
	fakeTool(t, "pbcopy", "exit 3")
	if err := Copy("x"); err == nil {
		t.Error("a tool that fails should be an error")
	}
}
