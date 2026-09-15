package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The vi pages: a screen forward and back, half a screen down and up.
func TestPagingThroughTheList(t *testing.T) {
	m, _ := newModel(t)
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 14}) // ten rows fit
	m = bulk(t, m, 40)
	page := m.listHeight()

	m = press(t, m, "ctrl+f")
	if m.cursor != page {
		t.Fatalf("ctrl+f: cursor = %d, want a full page of %d", m.cursor, page)
	}
	m = press(t, m, "ctrl+d")
	if m.cursor != page+page/2 {
		t.Errorf("ctrl+d: cursor = %d, want half a page further", m.cursor)
	}
	m = press(t, m, "ctrl+u")
	if m.cursor != page {
		t.Errorf("ctrl+u: cursor = %d, want half a page back", m.cursor)
	}
	m = press(t, m, "ctrl+b")
	if m.cursor != 0 {
		t.Errorf("ctrl+b: cursor = %d, want back to the top", m.cursor)
	}
}

// Paging past either end stops there rather than doing nothing.
func TestPagingStopsAtTheEnds(t *testing.T) {
	m, _ := newModel(t)
	for range 5 {
		m = press(t, m, "ctrl+f")
	}
	if m.cursor != len(m.tasks)-1 {
		t.Errorf("cursor = %d, want the last row", m.cursor)
	}
	for range 5 {
		m = press(t, m, "ctrl+b")
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want the first row", m.cursor)
	}
}

// longDesc is more description than any terminal will show at once.
func longDesc() string {
	var b strings.Builder
	for i := range 40 {
		b.WriteString("line " + itoa(int64(i)) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func TestPagingThroughTheDetailView(t *testing.T) {
	m, _ := newModel(t)
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 14})
	m = withDesc(t, m, longDesc())
	m = press(t, m, "enter")
	if m.mode != modeDetail {
		t.Fatal("enter should open the detail view")
	}
	if strings.Contains(m.View(), "line 39") {
		t.Fatalf("this terminal cannot show it all at once:\n%s", m.View())
	}
	if !strings.Contains(m.View(), "more") {
		t.Errorf("the hint should say there is more:\n%s", m.View())
	}

	for range 10 {
		m = press(t, m, "ctrl+f")
	}
	if m.mode != modeDetail {
		t.Fatalf("paging should not close the view, mode = %v", m.mode)
	}
	if !strings.Contains(m.View(), "line 39") {
		t.Errorf("paging should reach the end:\n%s", m.View())
	}
	for range 10 {
		m = press(t, m, "ctrl+b")
	}
	if !strings.Contains(m.View(), "#1") {
		t.Errorf("and back to the top:\n%s", m.View())
	}
}

// Line by line as well, as in the help: the same keys that move the list.
func TestScrollingTheDetailViewALineAtATime(t *testing.T) {
	m, _ := newModel(t)
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 14})
	m = withDesc(t, m, longDesc())
	m = press(t, m, "enter")
	first := m.View()
	m = press(t, m, "j")
	if m.mode != modeDetail {
		t.Fatalf("j should scroll rather than close, mode = %v", m.mode)
	}
	if m.View() == first {
		t.Error("and the view should have moved")
	}
	m = press(t, m, "k")
	if m.View() != first {
		t.Error("k should put it back")
	}
}

// Everything that is not a way of moving still closes it, and E still edits.
func TestTheDetailViewStillClosesOnAnythingElse(t *testing.T) {
	for _, k := range []string{"esc", "q", "e", "x", " "} {
		m, _ := newModel(t)
		m = press(t, m, "enter")
		m = press(t, m, k)
		if m.mode != modeList {
			t.Errorf("%q should close the detail view, mode = %v", k, m.mode)
		}
	}
}

// Reopening starts at the top: where you left off reading one task is not where
// you want to start on the next.
func TestReopeningTheDetailViewStartsAtTheTop(t *testing.T) {
	m, _ := newModel(t)
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 80, Height: 14})
	m = withDesc(t, m, longDesc())
	m = press(t, m, "enter")
	m = press(t, m, "ctrl+f")
	m = press(t, m, "esc")
	m = press(t, m, "enter")
	if !strings.Contains(m.View(), "#1") {
		t.Errorf("it should open at the top:\n%s", m.View())
	}
}

// ctrl+d is a page now, so it is no longer a way out; ctrl+c still is.
func TestCtrlDNoLongerQuits(t *testing.T) {
	m, _ := newModel(t)
	next, cmd := m.Update(keyMsg("ctrl+d"))
	if quits(t, next.(Model), cmd) {
		t.Error("ctrl+d should not quit")
	}
	if strings.Contains(next.(Model).View(), "again") {
		t.Error("nor arm a quit")
	}
	next, _ = m.Update(keyMsg("ctrl+c"))
	m = next.(Model)
	_, cmd = m.Update(keyMsg("ctrl+c"))
	if !quits(t, m, cmd) {
		t.Error("ctrl+c twice should still quit")
	}
}
