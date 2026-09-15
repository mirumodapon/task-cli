package dump

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// The YAML written here is deliberately the plainest shape there is: a list of
// mappings, one level deep, with every string double quoted. Quoting always
// costs a little readability and buys the whole class of bugs where a value
// that looks like a number, a date or a "yes" stops being a string. It is also
// what makes the reader below small enough to trust.

func writeYAML(w io.Writer, ds []dumpTask) error {
	for _, d := range ds {
		if _, err := fmt.Fprintf(w, "- id: %d\n", d.ID); err != nil {
			return err
		}
		fields := [][2]string{
			{"title", d.Title},
			{"desc", d.Desc},
			{"project", d.Project},
			{"due", d.Due},
			{"priority", d.Priority},
			{"done_at", d.DoneAt},
			{"deleted_at", d.DeletedAt},
			{"created_at", d.CreatedAt},
			{"updated_at", d.UpdatedAt},
		}
		for _, f := range fields {
			if f[1] == "" {
				continue
			}
			fmt.Fprintf(w, "  %s: %s\n", f[0], quote(f[1]))
		}
		if len(d.Tags) > 0 {
			quoted := make([]string, 0, len(d.Tags))
			for _, tag := range d.Tags {
				quoted = append(quoted, quote(tag))
			}
			fmt.Fprintf(w, "  tags: [%s]\n", strings.Join(quoted, ", "))
		}
	}
	return nil
}

// quote renders a string as a YAML double-quoted scalar, which is also a JSON
// string: the escaping rules are the ones strconv already knows.
func quote(s string) string { return strconv.Quote(s) }

// readYAML parses what writeYAML wrote, and only that. A stricter reader than a
// YAML library would be is the point: anything it does not recognise is said
// out loud rather than quietly read as something else.
func readYAML(r io.Reader) ([]dumpTask, error) {
	var out []dumpTask
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8<<20)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		switch {
		case strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#"):
			continue
		case strings.HasPrefix(line, "- "):
			out = append(out, dumpTask{})
			line = "  " + strings.TrimPrefix(line, "- ")
		case strings.HasPrefix(line, "  "):
			if len(out) == 0 {
				return nil, fmt.Errorf("line %d: a field before any task", n)
			}
		default:
			return nil, fmt.Errorf("line %d: not a task dump: %q", n, line)
		}
		key, value, ok := strings.Cut(strings.TrimSpace(line), ": ")
		if !ok {
			return nil, fmt.Errorf("line %d: %q is not \"key: value\"", n, strings.TrimSpace(line))
		}
		if err := assign(&out[len(out)-1], key, value); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func assign(d *dumpTask, key, value string) error {
	if key == "id" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("id %q is not a number", value)
		}
		d.ID = id
		return nil
	}
	if key == "tags" {
		list, err := unquoteList(value)
		if err != nil {
			return err
		}
		d.Tags = list
		return nil
	}
	s, err := strconv.Unquote(value)
	if err != nil {
		return fmt.Errorf("%s: %q is not a quoted string", key, value)
	}
	switch key {
	case "title":
		d.Title = s
	case "desc":
		d.Desc = s
	case "project":
		d.Project = s
	case "due":
		d.Due = s
	case "priority":
		d.Priority = s
	case "done_at":
		d.DoneAt = s
	case "deleted_at":
		d.DeletedAt = s
	case "created_at":
		d.CreatedAt = s
	case "updated_at":
		d.UpdatedAt = s
	default:
		return fmt.Errorf("unknown field %q", key)
	}
	return nil
}

func unquoteList(value string) ([]string, error) {
	inner := strings.TrimSpace(value)
	if !strings.HasPrefix(inner, "[") || !strings.HasSuffix(inner, "]") {
		return nil, fmt.Errorf("tags: %q is not a list", value)
	}
	inner = strings.TrimSpace(inner[1 : len(inner)-1])
	if inner == "" {
		return nil, nil
	}
	var out []string
	for _, part := range strings.Split(inner, ", ") {
		s, err := strconv.Unquote(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("tags: %q is not a quoted string", part)
		}
		out = append(out, s)
	}
	return out, nil
}
