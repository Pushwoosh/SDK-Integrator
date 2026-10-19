package web

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pushwoosh/SDK-Integrator/internal/recipe"
)

var creds = recipe.Credentials{ApplicationCode: "AAAAA-BBBBB", APIToken: "tok123"}

func mkProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func applied(t *testing.T, dir string) map[string]string {
	t.Helper()
	plan, err := Recipe{}.Plan(dir, creds)
	if err != nil {
		t.Fatal(err)
	}
	if err := recipe.Apply(dir, plan); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, c := range plan.Changes {
		data, _ := os.ReadFile(filepath.Join(dir, c.Path))
		out[c.Path] = string(data)
	}

	return out
}

func TestNextAppRouter(t *testing.T) {
	dir := mkProject(t, map[string]string{
		"package.json":   `{"dependencies":{"next":"15.0.0","react":"19.0.0"}}`,
		"pnpm-lock.yaml": "",
		"app/layout.tsx": "import type { Metadata } from 'next';\nimport './globals.css';\n\nexport default function RootLayout({ children }) {\n  return (\n    <html lang=\"en\">\n      <body className=\"x\">{children}</body>\n    </html>\n  );\n}\n",
	})
	plan, err := Recipe{}.Plan(dir, creds)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(plan.Command, " ") != "pnpm add web-push-notifications" {
		t.Fatalf("command %v", plan.Command)
	}
	files := applied(t, dir)
	layout := files["app/layout.tsx"]
	if !strings.Contains(layout, "import './globals.css';\nimport { PushwooshInit } from './pushwoosh-init';") {
		t.Fatalf("import not after the last import:\n%s", layout)
	}
	if !strings.Contains(layout, "<body className=\"x\">\n        <PushwooshInit />{children}") {
		t.Fatalf("component not mounted in body:\n%s", layout)
	}
	if c := files["app/pushwoosh-init.tsx"]; !strings.HasPrefix(c, "'use client';") || !strings.Contains(c, "apiToken: 'tok123'") {
		t.Fatalf("component:\n%s", c)
	}
	if !strings.Contains(files["public/pushwoosh-service-worker.js"], "importScripts") {
		t.Fatal("service worker missing")
	}

	if _, err := (Recipe{}).Plan(dir, creds); !errors.Is(err, ErrAlreadyIntegrated) {
		t.Fatalf("second run: %v", err)
	}
}

func TestViteMultilineImport(t *testing.T) {
	dir := mkProject(t, map[string]string{
		"package.json": `{"dependencies":{"react":"19"},"devDependencies":{"vite":"6"}}`,
		"src/main.tsx": "import {\n  StrictMode,\n} from 'react';\nimport './index.css';\n\ncreateRoot(root).render(<App />);\n",
	})
	main := applied(t, dir)["src/main.tsx"]
	if !strings.Contains(main, "import './index.css';\nimport { Pushwoosh } from 'web-push-notifications';") {
		t.Fatalf("import placed wrong:\n%s", main)
	}
	if !strings.HasSuffix(main, "applicationCode: 'AAAAA-BBBBB',\n  apiToken: 'tok123',\n}]);\n") {
		t.Fatalf("init missing:\n%s", main)
	}
}

func TestNuxtAndStatic(t *testing.T) {
	nuxt := mkProject(t, map[string]string{"package.json": `{"dependencies":{"nuxt":"3","web-push-notifications":"3"}}`})
	plan, err := Recipe{}.Plan(nuxt, creds)
	if err != nil || plan.Command != nil || plan.Changes[1].Path != "plugins/pushwoosh.client.ts" {
		t.Fatalf("nuxt plan %+v, err %v", plan, err)
	}

	static := mkProject(t, map[string]string{"index.html": "<html><head><title>x</title></head><body></body></html>"})
	files := applied(t, static)
	if !strings.Contains(files["index.html"], "Pushwoosh.push(['init'") || !strings.Contains(files["index.html"], "</script>\n</head>") {
		t.Fatalf("static:\n%s", files["index.html"])
	}
	if _, ok := files["pushwoosh-service-worker.js"]; !ok {
		t.Fatal("service worker not next to index.html")
	}
}

func TestUnsupportedAndStaleFile(t *testing.T) {
	if _, err := (Recipe{}).Plan(mkProject(t, map[string]string{"package.json": `{"dependencies":{"express":"4"}}`}), creds); err == nil {
		t.Fatal("express planned")
	}

	dir := mkProject(t, map[string]string{"index.html": "<head></head>"})
	plan, err := Recipe{}.Plan(dir, creds)
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<head>edited</head>"), 0o644)
	if err := recipe.Apply(dir, plan); err == nil {
		t.Fatal("applied over a file changed after planning")
	}
}
