package recipe

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyCreatesAndGuards(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("one\n"), 0o644)
	plan := &Plan{Changes: []Change{
		{Path: "a.txt", Before: "one\n", After: "one\ntwo\n"},
		{Path: "deep/new.txt", After: "new\n"},
	}}
	if err := Apply(dir, plan); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "deep", "new.txt")); string(data) != "new\n" {
		t.Fatalf("new file: %q", data)
	}
	// The plan is now stale: a.txt no longer matches Before.
	if err := Apply(dir, plan); err == nil {
		t.Fatal("applied over a changed file")
	}
}

func TestDiff(t *testing.T) {
	if d := Diff(Change{Path: "n.js", After: "x\ny\n"}); !strings.Contains(d, "(new file)") || !strings.Contains(d, "+ x\n+ y\n") {
		t.Fatalf("new file diff:\n%s", d)
	}
	d := Diff(Change{Path: "m.js", Before: "a\nb\n", After: "a\nadded\nb\n"})
	if !strings.Contains(d, "+ added") || strings.Contains(d, "- ") {
		t.Fatalf("insert diff:\n%s", d)
	}
}
