// Package clipboard puts text on the system clipboard by handing it to whatever
// tool the machine has, the way the editor package hands text to $EDITOR.
//
// Bubble Tea v1 has no clipboard of its own, and a library for it would be a
// fifth dependency for something the platform already ships.
package clipboard

import (
	"fmt"
	"os/exec"
	"strings"
)

// tools are the ways to reach a clipboard, in the order they are tried:
// the platform's own first, then the two X11 ones, then WSL's.
var tools = [][]string{
	{"pbcopy"},
	{"wl-copy"},
	{"xclip", "-selection", "clipboard"},
	{"xsel", "--clipboard", "--input"},
	{"clip.exe"},
}

// Copy puts text on the clipboard. Text goes in on stdin rather than as an
// argument: a task's description can be long, and arguments have a limit.
func Copy(text string) error {
	for _, t := range tools {
		path, err := exec.LookPath(t[0])
		if err != nil {
			continue
		}
		c := exec.Command(path, t[1:]...)
		c.Stdin = strings.NewReader(text)
		if err := c.Run(); err != nil {
			return fmt.Errorf("%s: %w", t[0], err)
		}
		return nil
	}
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t[0])
	}
	return fmt.Errorf("no clipboard tool found (looked for %s)", strings.Join(names, ", "))
}

// readers are the ways to read a clipboard, paired with the writers above.
var readers = [][]string{
	{"pbpaste"},
	{"wl-paste", "--no-newline"},
	{"xclip", "-selection", "clipboard", "-o"},
	{"xsel", "--clipboard", "--output"},
	{"powershell.exe", "-NoProfile", "-Command", "Get-Clipboard"},
}

// Paste returns what is on the clipboard.
func Paste() (string, error) {
	for _, t := range readers {
		path, err := exec.LookPath(t[0])
		if err != nil {
			continue
		}
		out, err := exec.Command(path, t[1:]...).Output()
		if err != nil {
			return "", fmt.Errorf("%s: %w", t[0], err)
		}
		return string(out), nil
	}
	names := make([]string, 0, len(readers))
	for _, t := range readers {
		names = append(names, t[0])
	}
	return "", fmt.Errorf("no clipboard tool found (looked for %s)", strings.Join(names, ", "))
}
