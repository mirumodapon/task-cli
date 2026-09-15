// Package dump moves tasks in and out of files. json and yaml round trip, so
// what one command writes another reads back; sql is what sqlite3 takes; txt is
// for people.
package dump

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"todo.mirumo.net/internal/project"
	"todo.mirumo.net/internal/task"
)

// Format is how a dump is written.
type Format int

const (
	JSON Format = iota
	YAML
	SQL
	Text
)

func (f Format) String() string {
	switch f {
	case YAML:
		return "yaml"
	case SQL:
		return "sql"
	case Text:
		return "txt"
	}
	return "json"
}

// Names lists the formats, for help text and error messages.
func Names() []string { return []string{"txt", "json", "yaml", "sql"} }

// FormatFor reads the name of a format.
func FormatFor(name string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "json":
		return JSON, nil
	case "yaml", "yml":
		return YAML, nil
	case "sql":
		return SQL, nil
	case "txt", "text":
		return Text, nil
	}
	return JSON, fmt.Errorf("unknown format %q (use %s)", name, strings.Join(Names(), ", "))
}

// FormatForFile guesses from a file's extension, so -o tasks.yaml needs no -f.
func FormatForFile(path string) (Format, bool) {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		return JSON, false
	}
	f, err := FormatFor(ext)
	return f, err == nil
}

// dumpTask is a task on its way to a file. Dates are written the way they were
// given — a date-only due date stays date-only — so a round trip changes
// nothing about what the task means.
type dumpTask struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Desc      string   `json:"desc,omitempty"`
	Project   string   `json:"project,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Due       string   `json:"due,omitempty"`
	Priority  string   `json:"priority,omitempty"`
	DoneAt    string   `json:"done_at,omitempty"`
	DeletedAt string   `json:"deleted_at,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

const (
	dateLayout     = "2006-01-02"
	dateTimeLayout = "2006-01-02 15:04"
)

func toDump(t task.Task) dumpTask {
	d := dumpTask{
		ID: t.ID, Title: t.Title, Desc: t.Desc, Project: t.Project, Tags: t.Tags,
		Priority:  t.Priority.String(),
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
	if t.Due != nil {
		layout := dateLayout
		if t.DueHasTime {
			layout = dateTimeLayout
		}
		d.Due = t.Due.Format(layout)
	}
	if t.DoneAt != nil {
		d.DoneAt = t.DoneAt.Format(time.RFC3339)
	}
	if t.DeletedAt != nil {
		d.DeletedAt = t.DeletedAt.Format(time.RFC3339)
	}
	return d
}

func (d dumpTask) toTask() (task.Task, error) {
	t := task.Task{ID: d.ID, Title: d.Title, Desc: d.Desc, Project: d.Project, Tags: d.Tags}
	var err error
	if t.Priority, err = task.ParsePriority(d.Priority); err != nil {
		return task.Task{}, fmt.Errorf("#%d: %w", d.ID, err)
	}
	if d.Due != "" {
		// The two layouts are tried rather than guessed at, so a date-only due
		// date comes back date-only.
		if due, err := time.ParseInLocation(dateTimeLayout, d.Due, time.Local); err == nil {
			t.Due, t.DueHasTime = &due, true
		} else {
			due, err := time.ParseInLocation(dateLayout, d.Due, time.Local)
			if err != nil {
				return task.Task{}, fmt.Errorf("#%d: unreadable due date %q", d.ID, d.Due)
			}
			t.Due = &due
		}
	}
	for _, p := range []struct {
		name string
		raw  string
		out  **time.Time
	}{{"done_at", d.DoneAt, &t.DoneAt}, {"deleted_at", d.DeletedAt, &t.DeletedAt}} {
		if p.raw == "" {
			continue
		}
		v, err := time.Parse(time.RFC3339, p.raw)
		if err != nil {
			return task.Task{}, fmt.Errorf("#%d: unreadable %s %q", d.ID, p.name, p.raw)
		}
		*p.out = &v
	}
	if t.CreatedAt, err = parseStamp(d.CreatedAt); err != nil {
		return task.Task{}, fmt.Errorf("#%d: unreadable created_at %q", d.ID, d.CreatedAt)
	}
	if t.UpdatedAt, err = parseStamp(d.UpdatedAt); err != nil {
		return task.Task{}, fmt.Errorf("#%d: unreadable updated_at %q", d.ID, d.UpdatedAt)
	}
	return t, nil
}

func parseStamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("missing")
	}
	return time.Parse(time.RFC3339, s)
}

// Write renders tasks in the given format.
func Write(w io.Writer, ts []task.Task, f Format) error {
	out := make([]dumpTask, 0, len(ts))
	for _, t := range ts {
		out = append(out, toDump(t))
	}
	switch f {
	case JSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	case YAML:
		return writeYAML(w, out)
	case SQL:
		return writeSQL(w, out)
	}
	return writeText(w, ts)
}

// Read takes back what Write put out. Only the two formats meant for machines
// can be: txt drops what it cannot show, and sql is a program for sqlite3 to
// run rather than a file to parse.
func Read(r io.Reader, f Format) ([]task.Task, error) {
	var ds []dumpTask
	switch f {
	case JSON:
		if err := json.NewDecoder(r).Decode(&ds); err != nil {
			return nil, fmt.Errorf("not a task dump: %w", err)
		}
	case YAML:
		var err error
		if ds, err = readYAML(r); err != nil {
			return nil, err
		}
	case SQL:
		return nil, fmt.Errorf("an sql dump is restored with sqlite3, not by importing it")
	default:
		return nil, fmt.Errorf("a txt dump is for reading, not for reading back; export json, yaml or sql to import it again")
	}
	out := make([]task.Task, 0, len(ds))
	for _, d := range ds {
		t, err := d.toTask()
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func writeText(w io.Writer, ts []task.Task) error {
	for i, t := range ts {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		status := "open"
		if t.Done() {
			status = "done " + t.DoneAt.Format(dateTimeLayout)
		}
		if t.Deleted() {
			status += ", deleted " + t.DeletedAt.Format(dateTimeLayout)
		}
		fmt.Fprintf(w, "#%d  %s\n", t.ID, t.Title)
		fmt.Fprintf(w, "  status    %s\n", status)
		if t.Due != nil {
			layout := dateLayout
			if t.DueHasTime {
				layout = dateTimeLayout
			}
			fmt.Fprintf(w, "  due       %s\n", t.Due.Format(layout))
		}
		if t.Priority != task.PriNone {
			fmt.Fprintf(w, "  priority  %s\n", t.Priority)
		}
		if t.Project != "" {
			fmt.Fprintf(w, "  project   %s  %s\n", project.Label(t.Project), t.Project)
		}
		if len(t.Tags) > 0 {
			fmt.Fprintf(w, "  tags      @%s\n", strings.Join(t.Tags, " @"))
		}
		fmt.Fprintf(w, "  created   %s\n", t.CreatedAt.Format(dateTimeLayout))
		if t.Desc != "" {
			fmt.Fprintln(w)
			for _, line := range strings.Split(t.Desc, "\n") {
				if line == "" {
					fmt.Fprintln(w)
					continue
				}
				fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
	return nil
}
