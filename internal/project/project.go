// Package project reads what a recipe needs to know about the directory it runs in.
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// ReadPackageJSON returns nil when the directory is not a Node project.
func ReadPackageJSON(dir string) *PackageJSON {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var p PackageJSON
	if json.Unmarshal(data, &p) != nil {
		return nil
	}

	return &p
}

func (p *PackageJSON) Has(name string) bool {
	if p == nil {
		return false
	}
	_, dep := p.Dependencies[name]
	_, dev := p.DevDependencies[name]

	return dep || dev
}

// AddCommand is the project's own package manager adding pkg, chosen by its lockfile.
func AddCommand(dir, pkg string) []string {
	switch {
	case Exists(dir, "pnpm-lock.yaml"):
		return []string{"pnpm", "add", pkg}
	case Exists(dir, "bun.lock"), Exists(dir, "bun.lockb"):
		return []string{"bun", "add", pkg}
	case Exists(dir, "yarn.lock"):
		return []string{"yarn", "add", pkg}
	default:
		return []string{"npm", "install", pkg}
	}
}

func Exists(dir string, parts ...string) bool {
	_, err := os.Stat(filepath.Join(append([]string{dir}, parts...)...))

	return err == nil
}

// FirstExisting returns the first relative path that exists, or "".
func FirstExisting(dir string, candidates ...string) string {
	for _, c := range candidates {
		if Exists(dir, c) {
			return c
		}
	}

	return ""
}

func Read(dir, rel string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, rel))

	return string(data), err
}
