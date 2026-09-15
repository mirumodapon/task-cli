package dump

import (
	"fmt"
	"io"
	"strings"
)

// writeSQL emits statements against the schema the store actually uses, so a
// dump is restored by feeding it to sqlite3 rather than by anything here
// interpreting it.
func writeSQL(w io.Writer, ds []dumpTask) error {
	if _, err := fmt.Fprintln(w, "BEGIN;"); err != nil {
		return err
	}
	for _, d := range ds {
		fmt.Fprintf(w, "INSERT INTO tasks (id, title, description, project, due, priority, done_at, deleted_at, created_at, updated_at)\n")
		fmt.Fprintf(w, "  VALUES (%d, %s, %s, %s, %s, %d, %s, %s, %s, %s);\n",
			d.ID, sqlString(d.Title), sqlString(d.Desc), sqlString(d.Project),
			sqlNull(d.Due), priorityNumber(d.Priority),
			sqlNull(d.DoneAt), sqlNull(d.DeletedAt),
			sqlString(d.CreatedAt), sqlString(d.UpdatedAt))
		for _, tag := range d.Tags {
			fmt.Fprintf(w, "INSERT OR IGNORE INTO tags (name) VALUES (%s);\n", sqlString(tag))
			fmt.Fprintf(w, "INSERT OR IGNORE INTO task_tags (task_id, tag_id) SELECT %d, id FROM tags WHERE name = %s;\n",
				d.ID, sqlString(tag))
		}
	}
	_, err := fmt.Fprintln(w, "COMMIT;")
	return err
}

// sqlString quotes a string for SQLite, which escapes a quote by doubling it.
func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func sqlNull(s string) string {
	if s == "" {
		return "NULL"
	}
	return sqlString(s)
}

// priorityNumber is what the column holds: the order matters to ORDER BY, so
// the word is turned back into the number the schema stores.
func priorityNumber(word string) int {
	switch word {
	case "low":
		return 1
	case "med":
		return 2
	case "high":
		return 3
	}
	return 0
}
