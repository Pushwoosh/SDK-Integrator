// Package recipe holds what every SDK recipe produces: a plan of file changes shown before anything is written.
package recipe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Change is one file the plan creates or rewrites; Before is empty for a new file.
type Change struct {
	Path   string
	Before string
	After  string
}

// Plan is everything an install does: file changes plus at most one package manager command.
type Plan struct {
	Stack   string
	Changes []Change
	// Command installs the SDK package, e.g. ["npm", "install", "web-push-notifications"]; empty when not needed.
	Command []string
	Notes   []string
}

// Recipe turns a detected project into a plan.
type Recipe interface {
	Name() string
	Detect(dir string) bool
	Plan(dir string, creds Credentials) (*Plan, error)
}

type Credentials struct {
	ApplicationCode string
	APIToken        string
}

// Apply writes the changes; it refuses to touch a file that changed since the plan was made.
func Apply(dir string, p *Plan) error {
	for _, c := range p.Changes {
		path := filepath.Join(dir, c.Path)
		current, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if string(current) != c.Before {
			return fmt.Errorf("%s changed after the plan was shown; run the integrator again", c.Path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(c.After), 0o644); err != nil {
			return err
		}
	}

	return nil
}

// Diff renders a change as added and removed lines, enough to review an insert-only edit.
func Diff(c Change) string {
	var b strings.Builder
	if c.Before == "" {
		fmt.Fprintf(&b, "+++ %s (new file)\n", c.Path)
		for _, l := range strings.Split(strings.TrimRight(c.After, "\n"), "\n") {
			b.WriteString("+ " + l + "\n")
		}

		return b.String()
	}

	fmt.Fprintf(&b, "~~~ %s\n", c.Path)
	before := strings.Split(c.Before, "\n")
	after := strings.Split(c.After, "\n")
	i, j := 0, 0
	for i < len(before) || j < len(after) {
		switch {
		case i < len(before) && j < len(after) && before[i] == after[j]:
			i++
			j++
		case j < len(after) && !contains(before[i:], after[j]):
			b.WriteString("+ " + after[j] + "\n")
			j++
		default:
			b.WriteString("- " + before[i] + "\n")
			i++
		}
	}

	return b.String()
}

func contains(lines []string, l string) bool {
	for _, x := range lines {
		if x == l {
			return true
		}
	}

	return false
}
