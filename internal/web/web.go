// Package web is the Web SDK recipe: service worker, package and init for Next.js, Nuxt, Vite and static sites.
package web

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Pushwoosh/SDK-Integrator/internal/project"
	"github.com/Pushwoosh/SDK-Integrator/internal/recipe"
)

const (
	pkg           = "web-push-notifications"
	serviceWorker = "pushwoosh-service-worker.js"
	workerBody    = "importScripts('https://cdn.pushwoosh.com/webpush/v3/pushwoosh-service-worker.js' + self.location.search);\n"
)

// ErrAlreadyIntegrated stops a second run from injecting a second init.
var ErrAlreadyIntegrated = errors.New("Pushwoosh is already initialized in this project")

var (
	importEnd  = regexp.MustCompile(`from\s+['"][^'"]+['"];?\s*$`)
	sideEffect = regexp.MustCompile(`^import\s+['"][^'"]+['"];?\s*$`)
	bodyOpen   = regexp.MustCompile(`<body[^>]*>`)
	headClose  = regexp.MustCompile(`(?i)</head>`)
)

type Recipe struct{}

func (Recipe) Name() string { return "web" }

func (Recipe) Detect(dir string) bool {
	return project.ReadPackageJSON(dir) != nil || project.Exists(dir, "index.html")
}

func (Recipe) Plan(dir string, creds recipe.Credentials) (*recipe.Plan, error) {
	pj := project.ReadPackageJSON(dir)
	switch {
	case pj.Has("next"):
		return planNext(dir, pj, creds)
	case pj.Has("nuxt"):
		return planNuxt(dir, pj, creds)
	case pj.Has("vite"):
		return planVite(dir, pj, creds)
	case pj == nil && project.Exists(dir, "index.html"):
		return planStatic(dir, creds)
	}

	return nil, fmt.Errorf("no supported web stack found (Next.js app router, Nuxt, Vite, static index.html); run with an AI agent instead: claude or cursor")
}

func initCall(creds recipe.Credentials, indent string) string {
	return fmt.Sprintf("%[1]spwInstance.push(['init', {\n%[1]s  applicationCode: '%[2]s',\n%[1]s  apiToken: '%[3]s',\n%[1]s}]);\n",
		indent, creds.ApplicationCode, creds.APIToken)
}

func worker(dir, publicDir string) recipe.Change {
	return recipe.Change{Path: filepath.Join(publicDir, serviceWorker), After: workerBody}
}

func install(dir string, pj *project.PackageJSON) []string {
	if pj.Has(pkg) {
		return nil
	}

	return project.AddCommand(dir, pkg)
}

func planNext(dir string, pj *project.PackageJSON, creds recipe.Credentials) (*recipe.Plan, error) {
	layout := project.FirstExisting(dir,
		"app/layout.tsx", "app/layout.jsx", "app/layout.js", "src/app/layout.tsx", "src/app/layout.jsx", "src/app/layout.js")
	if layout == "" {
		return nil, fmt.Errorf("Next.js pages router is not supported yet; run with an AI agent instead: claude or cursor")
	}
	src, err := project.Read(dir, layout)
	if err != nil {
		return nil, err
	}
	if strings.Contains(src, "PushwooshInit") {
		return nil, ErrAlreadyIntegrated
	}

	ext := ".jsx"
	if strings.HasSuffix(layout, ".tsx") {
		ext = ".tsx"
	}
	component := filepath.Join(filepath.Dir(layout), "pushwoosh-init"+ext)
	componentSrc := "'use client';\n\nimport { useEffect } from 'react';\n\nexport function PushwooshInit() {\n  useEffect(() => {\n" +
		"    import('" + pkg + "').then(({ Pushwoosh }) => {\n      const pwInstance = new Pushwoosh();\n" +
		initCall(creds, "      ") + "    });\n  }, []);\n\n  return null;\n}\n"

	withImport, err := addImport(src, "import { PushwooshInit } from './pushwoosh-init';")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", layout, err)
	}
	loc := bodyOpen.FindStringIndex(withImport)
	if loc == nil {
		return nil, fmt.Errorf("%s: no <body> tag to mount PushwooshInit in", layout)
	}
	after := withImport[:loc[1]] + "\n        <PushwooshInit />" + withImport[loc[1]:]

	return &recipe.Plan{
		Stack:   "Next.js (app router)",
		Command: install(dir, pj),
		Changes: []recipe.Change{
			worker(dir, "public"),
			{Path: component, After: componentSrc},
			{Path: layout, Before: src, After: after},
		},
	}, nil
}

func planNuxt(dir string, pj *project.PackageJSON, creds recipe.Credentials) (*recipe.Plan, error) {
	pluginDir := "plugins"
	if project.Exists(dir, "app", "app.vue") || project.Exists(dir, "app", "pages") {
		pluginDir = filepath.Join("app", "plugins")
	}
	plugin := filepath.Join(pluginDir, "pushwoosh.client.ts")
	if project.Exists(dir, plugin) {
		return nil, ErrAlreadyIntegrated
	}
	src := "import { Pushwoosh } from '" + pkg + "';\n\nexport default defineNuxtPlugin(() => {\n  const pwInstance = new Pushwoosh();\n" +
		initCall(creds, "  ") + "});\n"

	return &recipe.Plan{
		Stack:   "Nuxt",
		Command: install(dir, pj),
		Changes: []recipe.Change{worker(dir, "public"), {Path: plugin, After: src}},
	}, nil
}

func planVite(dir string, pj *project.PackageJSON, creds recipe.Credentials) (*recipe.Plan, error) {
	entry := project.FirstExisting(dir, "src/main.tsx", "src/main.ts", "src/main.jsx", "src/main.js", "src/index.tsx", "src/index.jsx")
	if entry == "" {
		return nil, fmt.Errorf("Vite project without src/main.* or src/index.*; run with an AI agent instead: claude or cursor")
	}
	src, err := project.Read(dir, entry)
	if err != nil {
		return nil, err
	}
	if strings.Contains(src, pkg) {
		return nil, ErrAlreadyIntegrated
	}
	withImport, err := addImport(src, "import { Pushwoosh } from '"+pkg+"';")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", entry, err)
	}
	after := strings.TrimRight(withImport, "\n") + "\n\nconst pwInstance = new Pushwoosh();\n" + initCall(creds, "")

	return &recipe.Plan{
		Stack:   "Vite",
		Command: install(dir, pj),
		Changes: []recipe.Change{worker(dir, "public"), {Path: entry, Before: src, After: after}},
	}, nil
}

func planStatic(dir string, creds recipe.Credentials) (*recipe.Plan, error) {
	src, err := project.Read(dir, "index.html")
	if err != nil {
		return nil, err
	}
	if strings.Contains(src, "pushwoosh-web-notifications.js") {
		return nil, ErrAlreadyIntegrated
	}
	loc := headClose.FindStringIndex(src)
	if loc == nil {
		return nil, fmt.Errorf("index.html has no </head>")
	}
	snippet := "  <script type=\"text/javascript\" src=\"https://cdn.pushwoosh.com/webpush/v3/pushwoosh-web-notifications.js\" async></script>\n" +
		"  <script type=\"text/javascript\">\n    var Pushwoosh = Pushwoosh || [];\n" +
		strings.ReplaceAll(initCall(creds, "    "), "pwInstance.push", "Pushwoosh.push") + "  </script>\n"

	return &recipe.Plan{
		Stack:   "static HTML",
		Changes: []recipe.Change{worker(dir, "."), {Path: "index.html", Before: src, After: src[:loc[0]] + snippet + src[loc[0]:]}},
		Notes:   []string{"Deploy pushwoosh-service-worker.js next to index.html, at the root of your domain."},
	}, nil
}

// addImport puts line after the last top-level import statement, which may span several lines.
func addImport(src, line string) (string, error) {
	lines := strings.Split(src, "\n")
	last, open := -1, false
	for i, l := range lines {
		switch {
		case open:
			if importEnd.MatchString(l) {
				last, open = i, false
			}
		case strings.HasPrefix(l, "import "):
			if importEnd.MatchString(l) || sideEffect.MatchString(l) {
				last = i
			} else {
				open = true
			}
		}
	}
	if open {
		return "", errors.New("unterminated import statement")
	}
	if last < 0 {
		return line + "\n" + src, nil
	}

	out := append(append(append([]string{}, lines[:last+1]...), line), lines[last+1:]...)

	return strings.Join(out, "\n"), nil
}
