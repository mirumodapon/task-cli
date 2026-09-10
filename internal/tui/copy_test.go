package tui

import (
	"strings"
	"testing"
)

// copied captures what a key put on the clipboard.
func copied(t *testing.T, m Model, key string) (Model, string) {
	t.Helper()
	var got string
	m.copy = func(text string) error { got = text; return nil }
	return press(t, m, key), got
}

func TestYCopiesTheTitle(t *testing.T) {
	m, _ := newModel(t)
	m, got := copied(t, m, "y")
	if got != "first" {
		t.Errorf("y copied %q, want the title alone", got)
	}
	if !strings.Contains(m.View(), "copied") {
		t.Errorf("it should say it happened:\n%s", m.View())
	}
}

// Y copies what the detail view shows, which is the whole of what a task is.
func TestShiftYCopiesTheWholeTask(t *testing.T) {
	m, _ := newModel(t)
	m = withDesc(t, m, "semi-skimmed\ntwo litres")
	_, got := copied(t, m, "Y")
	for _, want := range []string{"#1", "first", "status", "open", "@urgent", "semi-skimmed", "two litres"} {
		if !strings.Contains(got, want) {
			t.Errorf("the copy is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("it should be plain text, not what the screen looks like:\n%q", got)
	}
}

// A machine with no clipboard tool says so rather than looking like a key that
// did nothing.
func TestACopyThatFailsIsReported(t *testing.T) {
	m, _ := newModel(t)
	m.copy = func(string) error { return errFake }
	m = press(t, m, "y")
	if m.err == nil {
		t.Error("the failure should be recorded")
	}
}

func TestCopyingWithNothingUnderTheCursorDoesNothing(t *testing.T) {
	m, _ := newModel(t)
	called := false
	m.copy = func(string) error { called = true; return nil }
	m = press(t, m, "/")
	m = press(t, m, "zzzz") // matches nothing
	if len(m.tasks) != 0 {
		t.Fatalf("the search should have emptied the list, got %d", len(m.tasks))
	}
	m = press(t, m, "enter") // back to the list, filter kept
	m = press(t, m, "y")
	if called {
		t.Error("there is nothing to copy")
	}
}
