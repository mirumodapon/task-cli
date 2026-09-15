package cli

import (
	"fmt"
	"io"
	"os"

	"todo.mirumo.net/internal/argparse"
	"todo.mirumo.net/internal/dump"
	"todo.mirumo.net/internal/task"
)

func exportFlags() *argparse.Set {
	return argparse.New(append(filterFlags(),
		argparse.Spec{Long: "output", Short: "o", Kind: argparse.String,
			Usage: "Write to this file instead of standard output"},
		argparse.Spec{Long: "format", Short: "f", Kind: argparse.String,
			Usage: "txt, json, yaml or sql. Guessed from the output's extension, json otherwise"},
	)...)
}

// cmdExport writes tasks out. It takes the same filters ls does, but not its
// defaults: a dump that quietly left out everything done, deleted, or belonging
// to a project would be a poor thing to have trusted as a backup.
func (a *App) cmdExport(args []string) error {
	r, err := exportFlags().Parse(args)
	if err != nil {
		return err
	}
	if pos := r.Args(); len(pos) > 0 {
		return fmt.Errorf("export takes no positional arguments, got %q", pos[0])
	}
	f, err := a.filterFrom(r)
	if err != nil {
		return err
	}
	widen(&f, r)

	format := dump.JSON
	path := r.String("output")
	if guess, ok := dump.FormatForFile(path); ok {
		format = guess
	}
	if r.Changed("format") {
		if format, err = dump.FormatFor(r.String("format")); err != nil {
			return err
		}
	}

	ts, err := a.Store.List(f, a.Now())
	if err != nil {
		return err
	}

	out := io.Writer(a.Out)
	if path != "" {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		out = file
	}
	if err := dump.Write(out, ts, format); err != nil {
		return err
	}
	// Only when the dump went somewhere else: a word on stdout would be part of
	// the dump.
	if path != "" {
		fmt.Fprintf(a.Out, "wrote %d tasks to %s as %s\n", len(ts), path, format)
	}
	return nil
}

// widen turns the ls defaults into dump defaults: everything, unless a flag
// asked for less. A flag the user gave is left exactly as they gave it.
func widen(f *task.Filter, r *argparse.Result) {
	if !r.Changed("project") && !r.Bool("no-project") && !r.Bool("all-projects") {
		f.Project = nil
	}
	if !r.Bool("all") && !r.Bool("done") {
		f.IncludeDone = true
	}
	if !r.Bool("deleted") {
		f.IncludeDeleted = true
	}
}
