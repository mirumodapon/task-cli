package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"todo.mirumo.net/internal/argparse"
	"todo.mirumo.net/internal/store"
)

// pruneFlags are the filter flags minus the ones prune has no use for. It is
// always and only about done tasks, so -a, --done, and the ordering flags
// would each be a way of asking a question prune does not answer.
func pruneFlags() *argparse.Set {
	var specs []argparse.Spec
	for _, s := range filterFlags() {
		switch s.Long {
		case "all", "done", "sort", "reverse":
			continue
		}
		specs = append(specs, s)
	}
	return argparse.New(append(specs,
		argparse.Spec{Long: "force", Short: "f", Kind: argparse.Bool,
			Usage: "Destroy the tasks instead of putting them aside"},
		argparse.Spec{Long: "backup", Short: "b", Kind: argparse.OptionalString,
			Usage: "Write the removed tasks to a database file first; with no value, one named for the time beside the task database"},
	)...)
}

// cmdPrune clears out what is done. The filters are ls's, so "task ls --done"
// with the same flags is an exact preview of what prune will take.
func (a *App) cmdPrune(args []string) error {
	r, err := pruneFlags().Parse(args)
	if err != nil {
		return err
	}
	if pos := r.Args(); len(pos) > 0 {
		return fmt.Errorf("prune takes no positional arguments, got %q", pos[0])
	}
	f, err := a.filterFrom(r)
	if err != nil {
		return err
	}
	f.OnlyDone = true

	ts, err := a.Store.List(f, a.Now())
	if err != nil {
		return err
	}
	if len(ts) == 0 {
		fmt.Fprintln(a.Out, "nothing to prune")
		return nil
	}

	if r.Changed("backup") {
		path, err := a.backupPath(r)
		if err != nil {
			return err
		}
		// Written before anything is removed: a backup that failed must not be
		// discovered after the tasks it was meant to hold are gone.
		if err := store.Snapshot(path, ts); err != nil {
			return fmt.Errorf("writing the backup: %w", err)
		}
		fmt.Fprintf(a.Out, "wrote %d tasks to %s\n", len(ts), path)
	}

	force := r.Bool("force")
	for _, t := range ts {
		if force {
			if err := a.Store.Delete(t.ID); err != nil {
				return fmt.Errorf("#%d: %w", t.ID, err)
			}
			fmt.Fprintf(a.Out, "destroyed #%d: %s\n", t.ID, t.Title)
			continue
		}
		if err := a.Store.SetDeleted(t.ID, true, a.Now()); err != nil {
			return fmt.Errorf("#%d: %w", t.ID, err)
		}
		fmt.Fprintf(a.Out, "deleted #%d: %s\n", t.ID, t.Title)
	}
	return nil
}

// backupPath reads -b. A value is the file to write, taken as given. With no
// value the name is made up: the time, beside the database the tasks came from,
// which is ~/.todo unless --db or $TASK_DB moved it. Putting a backup beside
// its own database keeps the two together wherever the database lives.
func (a *App) backupPath(r *argparse.Result) (string, error) {
	if v, has := r.Optional("backup"); has {
		return v, nil
	}
	if a.DBPath == "" {
		return "", errors.New("-b needs a path: this build has no database file to put one beside")
	}
	name := a.Now().Format("20060102T150405") + "_db.sqlite"
	return filepath.Join(filepath.Dir(a.DBPath), name), nil
}
