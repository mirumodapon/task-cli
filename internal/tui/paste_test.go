package tui

import (
	"strings"
	"testing"
)

// pasting arms the clipboard with text and presses p.
func pasting(t *testing.T, m Model, text string, err error) Model {
	t.Helper()
	m.paste = func() (string, error) { return text, err }
	return press(t, m, "p")
}

// p turns whatever is on the clipboard into a new task, with the form open so
// the rest of the fields — and the title itself — can still be changed.
func TestPOpensTheFormOnTheClipboard(t *testing.T) {
	m, _ := newModel(t)
	m = pasting(t, m, "renew the passport", nil)
	if m.mode != modeForm {
		t.Fatalf("p should open the form, mode = %v", m.mode)
	}
	if m.form.editing {
		t.Error("it should be a new task, not an edit")
	}
	if !strings.Contains(m.View(), "renew the passport") {
		t.Errorf("the title should be filled in:\n%s", m.View())
	}
}

// A title is one line. Pasting a paragraph keeps every word of it rather than
// dropping what did not fit on the first line.
func TestPastingSeveralLinesBecomesOneTitle(t *testing.T) {
	m, _ := newModel(t)
	m = pasting(t, m, "  renew\nthe   passport \n\n", nil)
	if got := m.form.inputs[fieldTitle].Value(); got != "renew the passport" {
		t.Errorf("title = %q", got)
	}
}

func TestPastingNothingSaysSo(t *testing.T) {
	m, _ := newModel(t)
	m = pasting(t, m, "   \n", nil)
	if m.mode != modeList {
		t.Errorf("there is nothing to make a task of, mode = %v", m.mode)
	}
	if !strings.Contains(m.View(), "clipboard is empty") {
		t.Errorf("it should say why nothing happened:\n%s", m.View())
	}
}

func TestAPasteThatFailsIsReported(t *testing.T) {
	m, _ := newModel(t)
	m = pasting(t, m, "", errFake)
	if m.err == nil {
		t.Error("the failure should be recorded")
	}
	if m.mode != modeList {
		t.Errorf("and no form opened, mode = %v", m.mode)
	}
}

// What p opens is an ordinary form: enter saves it like any other.
func TestAPastedTaskSaves(t *testing.T) {
	m, s := newModel(t)
	m = pasting(t, m, "renew the passport", nil)
	m = press(t, m, "enter")
	ts, err := s.List(DefaultStart().Filter, refTime())
	if err != nil {
		t.Fatal(err)
	}
	for _, ti := range ts {
		if ti.Title == "renew the passport" {
			return
		}
	}
	t.Errorf("the pasted task should have been saved: %v", ts)
}
