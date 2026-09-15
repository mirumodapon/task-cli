package cli

import (
	"fmt"
	"os"

	"todo.mirumo.net/internal/argparse"
	"todo.mirumo.net/internal/dump"
)

func importFlags() *argparse.Set {
	return argparse.New(
		argparse.Spec{Long: "format", Short: "f", Kind: argparse.String,
			Usage: "json or yaml. Guessed from the file's extension, json otherwise"},
	)
}

// cmdImport reads a dump back in. The tasks are added, not restored over: ids
// belong to the list that gave them out, and two lists have no reason to agree
// about which task is #3.
func (a *App) cmdImport(args []string) error {
	r, err := importFlags().Parse(args)
	if err != nil {
		return err
	}
	pos := r.Args()
	if len(pos) > 1 {
		return fmt.Errorf("import takes one file, or none to read standard input; got %d", len(pos))
	}

	format := dump.JSON
	var path string
	if len(pos) == 1 && pos[0] != "-" {
		path = pos[0]
		if guess, ok := dump.FormatForFile(path); ok {
			format = guess
		}
	}
	if r.Changed("format") {
		if format, err = dump.FormatFor(r.String("format")); err != nil {
			return err
		}
	}

	in := a.In
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		in = file
	}
	if in == nil {
		return fmt.Errorf("nothing to read: give a file, or pipe a dump in")
	}

	// Everything is parsed before anything is written, so a file that turns out
	// not to be a dump leaves the list as it was.
	ts, err := dump.Read(in, format)
	if err != nil {
		return err
	}
	now := a.Now()
	for i, t := range ts {
		t.ID = 0
		if t.CreatedAt.IsZero() {
			t.CreatedAt = now
		}
		if t.UpdatedAt.IsZero() {
			t.UpdatedAt = now
		}
		if _, err := a.Store.Add(t); err != nil {
			return fmt.Errorf("after %d of %d: %w", i, len(ts), err)
		}
	}
	fmt.Fprintf(a.Out, "imported %d %s\n", len(ts), plural(len(ts), "task", "tasks"))
	return nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
