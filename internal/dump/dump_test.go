package dump

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"todo.mirumo.net/internal/task"
)

func ref() time.Time { return time.Date(2026, 8, 29, 15, 0, 0, 0, time.Local) }

func sample() []task.Task {
	due := time.Date(2026, 9, 12, 0, 0, 0, 0, time.Local)
	timed := time.Date(2026, 9, 1, 15, 30, 0, 0, time.Local)
	done := ref()
	return []task.Task{
		{
			ID: 1, Title: `buy "milk"`, Desc: "semi-skimmed\ntwo litres",
			Project: "/p/home", Tags: []string{"shopping", "chores"},
			Due: &due, Priority: task.PriHigh,
			CreatedAt: ref(), UpdatedAt: ref(),
		},
		{
			ID: 7, Title: "it's done", Due: &timed, DueHasTime: true,
			DoneAt: &done, CreatedAt: ref(), UpdatedAt: ref(),
		},
	}
}

func TestFormatForName(t *testing.T) {
	for _, name := range []string{"txt", "json", "yaml", "sql"} {
		if _, err := FormatFor(name); err != nil {
			t.Errorf("FormatFor(%q): %v", name, err)
		}
	}
	if _, err := FormatFor("xml"); err == nil {
		t.Error("an unknown format should be an error")
	}
	if _, err := FormatFor("YAML"); err != nil {
		t.Error("the name should not be case sensitive")
	}
}

// The extension is a good guess at what someone meant by -o tasks.yaml.
func TestFormatForFile(t *testing.T) {
	cases := map[string]Format{
		"/tmp/tasks.json": JSON,
		"tasks.yaml":      YAML,
		"tasks.yml":       YAML,
		"tasks.sql":       SQL,
		"tasks.txt":       Text,
	}
	for path, want := range cases {
		got, ok := FormatForFile(path)
		if !ok || got != want {
			t.Errorf("FormatForFile(%q) = %v, %v", path, got, ok)
		}
	}
	if _, ok := FormatForFile("tasks"); ok {
		t.Error("no extension is nothing to go on")
	}
}

// What Write puts out, Read takes back: that is the whole point of a dump.
func TestJSONAndYAMLRoundTrip(t *testing.T) {
	for _, f := range []Format{JSON, YAML} {
		t.Run(f.String(), func(t *testing.T) {
			var b bytes.Buffer
			if err := Write(&b, sample(), f); err != nil {
				t.Fatal(err)
			}
			got, err := Read(strings.NewReader(b.String()), f)
			if err != nil {
				t.Fatalf("Read: %v\n%s", err, b.String())
			}
			want := sample()
			if len(got) != len(want) {
				t.Fatalf("got %d tasks, want %d:\n%s", len(got), len(want), b.String())
			}
			for i := range want {
				assertSame(t, got[i], want[i], b.String())
			}
		})
	}
}

func assertSame(t *testing.T, got, want task.Task, raw string) {
	t.Helper()
	if got.ID != want.ID || got.Title != want.Title || got.Desc != want.Desc {
		t.Errorf("= %+v\nwant %+v\nfrom:\n%s", got, want, raw)
	}
	if got.Project != want.Project || strings.Join(got.Tags, ",") != strings.Join(want.Tags, ",") {
		t.Errorf("project or tags differ: %+v vs %+v", got, want)
	}
	if got.Priority != want.Priority {
		t.Errorf("priority = %v, want %v", got.Priority, want.Priority)
	}
	switch {
	case (got.Due == nil) != (want.Due == nil):
		t.Errorf("due = %v, want %v", got.Due, want.Due)
	case got.Due != nil && (!got.Due.Equal(*want.Due) || got.DueHasTime != want.DueHasTime):
		t.Errorf("due = %v (hasTime %v), want %v (%v)", got.Due, got.DueHasTime, want.Due, want.DueHasTime)
	}
	if (got.DoneAt == nil) != (want.DoneAt == nil) {
		t.Errorf("done_at = %v, want %v", got.DoneAt, want.DoneAt)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, want.CreatedAt)
	}
}

// Text is for reading, not for reading back.
func TestTextIsReadable(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, sample(), Text); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{`buy "milk"`, "#1", "semi-skimmed", "@shopping", "2026-09-12", "high", "it's done"} {
		if !strings.Contains(s, want) {
			t.Errorf("the text dump is missing %q:\n%s", want, s)
		}
	}
	if _, err := Read(strings.NewReader(s), Text); err == nil {
		t.Error("text should say it cannot be read back, rather than half doing it")
	}
}

// SQL is what sqlite3 will take, so the quoting has to survive a quote.
func TestSQLQuotesAndCarriesTags(t *testing.T) {
	var b bytes.Buffer
	if err := Write(&b, sample(), SQL); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if !strings.Contains(s, "'it''s done'") {
		t.Errorf("a quote in a title should be doubled:\n%s", s)
	}
	for _, want := range []string{"BEGIN;", "COMMIT;", "INSERT INTO tasks", "task_tags", "'shopping'"} {
		if !strings.Contains(s, want) {
			t.Errorf("the SQL dump is missing %q:\n%s", want, s)
		}
	}
	if _, err := Read(strings.NewReader(s), SQL); err == nil {
		t.Error("SQL is for sqlite3 to read, and should say so")
	}
}

func TestReadRejectsSomethingElse(t *testing.T) {
	if _, err := Read(strings.NewReader("not a dump at all"), JSON); err == nil {
		t.Error("a file that is not a dump should be an error")
	}
	if _, err := Read(strings.NewReader("hello: there\n"), YAML); err == nil {
		t.Error("YAML that is not a list of tasks should be an error")
	}
}

func TestWriteNothing(t *testing.T) {
	for _, f := range []Format{JSON, YAML, SQL, Text} {
		var b bytes.Buffer
		if err := Write(&b, nil, f); err != nil {
			t.Errorf("%v: %v", f, err)
		}
	}
	var b bytes.Buffer
	if err := Write(&b, nil, JSON); err != nil {
		t.Fatal(err)
	}
	got, err := Read(strings.NewReader(b.String()), JSON)
	if err != nil || len(got) != 0 {
		t.Errorf("an empty dump should read back as no tasks: %v, %v", got, err)
	}
}
