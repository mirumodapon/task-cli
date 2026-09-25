package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"todo.mirumo.net/internal/task"
)

// Snapshot writes tasks to a new database at path. What it produces is an
// ordinary task database rather than a dump: "task --db <path> ls" reads it,
// which is the whole point of a backup you might one day need in a hurry.
//
// Ids are kept. A backup whose tasks were renumbered could not be compared
// with the list it came from, and comparing is what you open a backup to do.
func Snapshot(path string, ts []task.Task) error {
	// Checked before opening, because opening creates. SQLite would happily add
	// rows to a database that is already there, and a backup folded into an
	// older backup is the one failure a backup must not have.
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	s, err := openSQLite(path)
	if err != nil {
		return err
	}
	defer s.Close()
	for _, t := range ts {
		if err := s.insertVerbatim(t); err != nil {
			return err
		}
	}
	return nil
}

// insertVerbatim writes one task exactly as given. Add is the wrong tool here:
// it assigns a fresh id and has no interest in deleted_at, both of which a
// backup has to carry.
func (s *sqlStore) insertVerbatim(t task.Task) error {
	if _, err := s.db.Exec(
		`INSERT INTO tasks (id, title, description, project, due, priority, done_at, deleted_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Desc, t.Project, dueVal(t.Due, t.DueHasTime), int(t.Priority),
		tsVal(t.DoneAt), tsVal(t.DeletedAt),
		t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339)); err != nil {
		return err
	}
	return s.setTags(t.ID, t.Tags)
}
