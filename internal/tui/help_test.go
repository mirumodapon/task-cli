package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func helpOn(t *testing.T, w, h int) Model {
	t.Helper()
	m, _ := newModel(t)
	m, _ = send(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	return press(t, m, "?")
}

// The last row of the help is the one most likely to be cut, and the one that
// says how to leave.
const lastHelpRow = "ctrl+c / ctrl+d"

func TestTheHelpFitsATallTerminal(t *testing.T) {
	m := helpOn(t, 80, 40)
	v := m.View()
	for _, r := range helpRows {
		if !strings.Contains(v, r[0]) {
			t.Errorf("the help is missing %q:\n%s", r[0], v)
		}
	}
	if strings.Contains(v, "more") {
		t.Errorf("nothing is cut, so nothing should say so:\n%s", v)
	}
}

// A short terminal cannot show every key at once, so it says so rather than
// quietly dropping the ones that did not fit.
func TestAShortTerminalSaysThereIsMore(t *testing.T) {
	m := helpOn(t, 80, 14)
	v := m.View()
	if strings.Contains(v, lastHelpRow) {
		t.Fatalf("this terminal cannot fit it all:\n%s", v)
	}
	if !strings.Contains(v, "more") {
		t.Errorf("it should say that there is more:\n%s", v)
	}
	if len(strings.Split(v, "\n")) != 14 {
		t.Errorf("the frame should still hold:\n%s", v)
	}
}

func TestScrollingReachesTheLastKey(t *testing.T) {
	m := helpOn(t, 80, 14)
	for range len(helpRows) {
		m = press(t, m, "j")
	}
	if m.mode != modeHelp {
		t.Fatalf("j should scroll rather than close, mode = %v", m.mode)
	}
	if !strings.Contains(m.View(), lastHelpRow) {
		t.Errorf("scrolling should reach the end:\n%s", m.View())
	}
	// And stop there.
	if !strings.Contains(m.View(), helpRows[len(helpRows)-1][1]) {
		t.Errorf("the bottom row should stay in view:\n%s", m.View())
	}
	for range len(helpRows) {
		m = press(t, m, "k")
	}
	if !strings.Contains(m.View(), helpRows[0][1]) {
		t.Errorf("k should scroll back to the top:\n%s", m.View())
	}
}

func TestAnyOtherKeyStillCloses(t *testing.T) {
	for _, k := range []string{"esc", "?", "q", "x"} {
		m := helpOn(t, 80, 14)
		m = press(t, m, k)
		if m.mode != modeList {
			t.Errorf("%q should close the help, mode = %v", k, m.mode)
		}
	}
}

// Opening it again starts at the top: where you left off scrolling is not
// where you want to start reading.
func TestReopeningStartsAtTheTop(t *testing.T) {
	m := helpOn(t, 80, 14)
	m = press(t, m, "j")
	m = press(t, m, "j")
	m = press(t, m, "esc")
	m = press(t, m, "?")
	if !strings.Contains(m.View(), helpRows[0][1]) {
		t.Errorf("it should open at the top:\n%s", m.View())
	}
}

// The list has grown past the point where a flat column is worth reading, so it
// is grouped by what the keys are for.
func TestTheHelpIsGrouped(t *testing.T) {
	v := helpOn(t, 80, 60).View()
	for _, want := range []string{"Moving", "Looking", "Changing", "Clipboard", "Finding", "Leaving"} {
		if !strings.Contains(v, want) {
			t.Errorf("the help should have a %q group:\n%s", want, v)
		}
	}
	// Every binding is still in there, under one of them.
	for _, r := range helpRows {
		if !strings.Contains(v, r[0]) {
			t.Errorf("the help lost %q:\n%s", r[0], v)
		}
	}
	// A heading is not a key: it has nothing in the second column.
	moving := strings.Index(v, "Moving")
	jk := strings.Index(v, "j / k")
	if moving < 0 || jk < moving {
		t.Errorf("the group should come before the keys in it:\n%s", v)
	}
}

// Grouping adds lines, so the scrolling has to carry the headings with them.
func TestScrollingPastAGrouping(t *testing.T) {
	m := helpOn(t, 80, 14)
	for range 60 {
		m = press(t, m, "j")
	}
	v := m.View()
	if !strings.Contains(v, lastHelpRow) {
		t.Errorf("scrolling should still reach the end:\n%s", v)
	}
	if !strings.Contains(v, "Leaving") {
		t.Errorf("and the heading of the group it is in:\n%s", v)
	}
}

// The help is inside the same frame as everything else: a description too long
// for the terminal is cut, not wrapped into the row below it.
func TestTheHelpStaysInsideTheFrame(t *testing.T) {
	for _, size := range [][2]int{{60, 20}, {70, 40}, {100, 24}} {
		m := helpOn(t, size[0], size[1])
		rows := strings.Split(m.View(), "\n")
		if len(rows) != size[1] {
			t.Errorf("%dx%d: the screen is %d rows", size[0], size[1], len(rows))
		}
		for _, l := range rows {
			if lipgloss.Width(l) > size[0] {
				t.Errorf("%dx%d: a line is %d columns: %q", size[0], size[1], lipgloss.Width(l), l)
			}
		}
	}
}
