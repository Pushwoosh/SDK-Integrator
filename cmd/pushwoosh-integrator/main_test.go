package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDirty(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	plain := t.TempDir()
	if dirty(plain) {
		t.Fatal("a directory outside git counted as dirty")
	}

	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	if dirty(repo) {
		t.Fatal("an empty repository counted as dirty")
	}
	_ = os.WriteFile(filepath.Join(repo, "x"), []byte("x"), 0o644)
	if !dirty(repo) {
		t.Fatal("an untracked file did not count as dirty")
	}
}
