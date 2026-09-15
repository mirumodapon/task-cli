package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"todo.mirumo.net/internal/datearg"
	"todo.mirumo.net/internal/editor"
	"todo.mirumo.net/internal/project"
	"todo.mirumo.net/internal/task"
	"todo.mirumo.net/internal/taskfile"
)

const stampLayout = "2006-01-02 15:04"

// detailRows lists what a task is, one label and value per row. Empty fields are
// left out: a column of blank labels buries the ones that say something. The
// labels and the order match task details, so the two views read the same.
func (m Model) detailRows(t task.Task) [][2]string {
	status := "open"
	if t.Done() {
		status = "done " + t.DoneAt.Format(stampLayout)
	}
	rows := [][2]string{{"status", status}}
	// Its own row rather than a clause on the status: being deleted is a
	// separate fact from being finished, and a task can be both.
	if t.Deleted() {
		rows = append(rows, [2]string{"deleted", t.DeletedAt.Format(stampLayout)})
	}
	if t.Due != nil {
		layout := "2006-01-02"
		if t.DueHasTime {
			layout = stampLayout
		}
		rows = append(rows, [2]string{"due", fmt.Sprintf("%s  (%s)",
			t.Due.Format(layout), datearg.Remaining(*t.Due, t.DueHasTime, m.now()))})
	}
	if t.Priority != task.PriNone {
		rows = append(rows, [2]string{"priority", t.Priority.Marks() + " " + t.Priority.String()})
	}
	if t.Project != "" {
		rows = append(rows, [2]string{"project", project.Label(t.Project) + "  " + t.Project})
	}
	if len(t.Tags) > 0 {
		rows = append(rows, [2]string{"tags", "@" + strings.Join(t.Tags, " @")})
	}
	rows = append(rows, [2]string{"created", t.CreatedAt.Format(stampLayout)})
	if !t.UpdatedAt.Equal(t.CreatedAt) {
		rows = append(rows, [2]string{"updated", t.UpdatedAt.Format(stampLayout)})
	}
	return rows
}

// editTaskCmd hands the whole task to the editor as a text file. The result
// comes back as an editedMsg, so the write still happens in a cmd like every
// other one.
func (m Model) editTaskCmd(t task.Task) tea.Cmd {
	now := m.now()
	return m.edit(taskfile.Format(t), func(text, path string, err error) tea.Msg {
		if err != nil {
			editor.Discard(path)
			return errMsg{err}
		}
		next, err := taskfile.Parse(text, t, now)
		if err != nil {
			// The file stays where it is: a typo in one field is no reason to
			// throw away the paragraph underneath it.
			if path != "" {
				return errMsg{fmt.Errorf("%w — your edit is kept at %s", err, path)}
			}
			return errMsg{err}
		}
		editor.Discard(path)
		next.UpdatedAt = now
		return editedMsg{next}
	})
}

// detailFit is how many lines of the task fit: the frame less the blank line
// above the hint and the hint itself.
func (m Model) detailFit() int { return max(1, m.height-2) }

// updateDetail handles keys while a task is open. Moving scrolls it, E edits it,
// and everything else closes it — the same bargain the help makes, because a
// description can be longer than the screen and still has to be readable.
func (m Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	t, ok := m.current()
	if !ok {
		m.mode = modeList
		return m, nil
	}
	lines := len(m.detailLines(t))
	switch msg.String() {
	case "E":
		return m, m.editTaskCmd(t)
	case "j", "down", "ctrl+n":
		m.detailOffset++
	case "k", "up", "ctrl+p":
		m.detailOffset--
	case "ctrl+f", "pgdown":
		m.detailOffset += m.detailFit()
	case "ctrl+b", "pgup":
		m.detailOffset -= m.detailFit()
	case "ctrl+d":
		m.detailOffset += max(1, m.detailFit()/2)
	case "ctrl+u":
		m.detailOffset -= max(1, m.detailFit()/2)
	case "g":
		m.detailOffset = 0
	case "G":
		m.detailOffset = lines
	default:
		m.mode = modeList
		return m, nil
	}
	m.detailOffset = max(0, min(m.detailOffset, lines-m.detailFit()))
	return m, nil
}

var (
	detailHint       = styleHint.Render("E edit the task in $EDITOR · any other key goes back")
	detailScrollHint = styleHint.Render("j/k for more · E edit in $EDITOR · any other key goes back")
)

// detailLines renders the whole task as lines. Scrolling then works on lines,
// so a wrapped description scrolls by what is on screen rather than by what it
// was before it was wrapped.
func (m Model) detailLines(t task.Task) []string {
	rows := m.detailRows(t)
	var w int
	for _, r := range rows {
		w = max(w, lipgloss.Width(r[0]))
	}
	out := []string{fmt.Sprintf("#%d  %s", t.ID, t.Title), ""}
	for _, r := range rows {
		out = append(out, "  "+styleDim.Render(pad(r[0], w))+"  "+r[1])
	}
	out = append(out, "")
	if t.Desc == "" {
		return append(out, styleDim.Render("  No description"))
	}
	// The description is wrapped to the frame: a long line would otherwise wrap
	// itself and push the lines below it off the bottom of the screen.
	for _, line := range strings.Split(t.Desc, "\n") {
		if line == "" {
			out = append(out, "")
			continue
		}
		// Width pads every rendered line out to the full frame, so the padding is
		// trimmed back off: trailing spaces are invisible until something copies them.
		for _, wrapped := range strings.Split(styleDescBody.Width(max(3, m.width)).Render(line), "\n") {
			out = append(out, strings.TrimRight(wrapped, " "))
		}
	}
	return out
}

func (m Model) viewDetail() string {
	t, ok := m.current()
	if !ok {
		return m.viewList()
	}
	lines := m.detailLines(t)
	fit := m.detailFit()
	scrolls := len(lines) > fit
	if scrolls {
		start := min(m.detailOffset, len(lines)-fit)
		lines = lines[start : start+fit]
	}
	hint := detailHint
	if scrolls {
		hint = detailScrollHint
	}
	return m.screen(strings.Join(lines, "\n")+"\n", m.hint(hint))
}

// editorFunc hands text to an editor and returns the cmd that produces the
// result. apply turns that result into the msg Update will see; it is handed
// the file's path as well, so it can leave the text on disk when it cannot make
// sense of what came back.
type editorFunc func(text string, apply func(text, path string, err error) tea.Msg) tea.Cmd

// execEditor is the real one. Bubble Tea has to give the terminal up while the
// editor owns it, which is what tea.ExecProcess does: it suspends the
// interface, runs the command, and restores the screen afterwards.
func execEditor(text string, apply func(text, path string, err error) tea.Msg) tea.Cmd {
	path, err := editor.WriteTemp("task", text)
	if err != nil {
		return func() tea.Msg { return apply("", "", err) }
	}
	return tea.ExecProcess(editor.Command(path), func(runErr error) tea.Msg {
		if runErr != nil {
			return apply("", path, runErr)
		}
		edited, err := editor.Read(path)
		return apply(edited, path, err)
	})
}

// copyText renders a task as plain text for the clipboard: the same facts the
// detail view shows, without the styling or the frame around them.
func (m Model) copyText(t task.Task) string {
	rows := m.detailRows(t)
	var w int
	for _, r := range rows {
		w = max(w, lipgloss.Width(r[0]))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "#%d  %s\n", t.ID, t.Title)
	for _, r := range rows {
		b.WriteString(pad(r[0], w) + "  " + r[1] + "\n")
	}
	if t.Desc != "" {
		b.WriteString("\n" + t.Desc + "\n")
	}
	return b.String()
}
