package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddCommandFollowsLockfile(t *testing.T) {
	for lock, want := range map[string]string{
		"pnpm-lock.yaml":    "pnpm add x",
		"bun.lock":          "bun add x",
		"bun.lockb":         "bun add x",
		"yarn.lock":         "yarn add x",
		"package-lock.json": "npm install x",
		"":                  "npm install x",
	} {
		dir := t.TempDir()
		if lock != "" {
			_ = os.WriteFile(filepath.Join(dir, lock), nil, 0o644)
		}
		if got := strings.Join(AddCommand(dir, "x"), " "); got != want {
			t.Errorf("%q: got %q, want %q", lock, got, want)
		}
	}
}

func TestPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if ReadPackageJSON(dir) != nil {
		t.Fatal("package.json found in an empty directory")
	}
	var none *PackageJSON
	if none.Has("next") {
		t.Fatal("nil package.json has a dependency")
	}

	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"dependencies":{"next":"15"},"devDependencies":{"vite":"6"}}`), 0o644)
	pj := ReadPackageJSON(dir)
	if !pj.Has("next") || !pj.Has("vite") || pj.Has("nuxt") {
		t.Fatalf("Has wrong for %+v", pj)
	}

	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{broken`), 0o644)
	if ReadPackageJSON(dir) != nil {
		t.Fatal("broken package.json parsed")
	}
}

func TestFirstExisting(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "src", "app"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "src", "app", "layout.tsx"), nil, 0o644)
	if got := FirstExisting(dir, "app/layout.tsx", "src/app/layout.tsx"); got != "src/app/layout.tsx" {
		t.Fatalf("got %q", got)
	}
	if FirstExisting(dir, "nope") != "" {
		t.Fatal("found a missing file")
	}
}
