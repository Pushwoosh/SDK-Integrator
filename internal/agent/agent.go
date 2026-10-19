// Package agent installs the Pushwoosh plugin and MCP server into an AI coding agent.
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	// MCPURL is the Pushwoosh MCP server; the agent signs in to it through the browser (OAuth).
	MCPURL      = "https://rpc-api.svc-nue.pushwoosh.com/mcp"
	marketplace = "Pushwoosh/SDK-Integrator"
	plugin      = "pushwoosh@pushwoosh-sdk"
)

// Install sets the agent up and returns what the user types next.
func Install(name, dir, code string) (string, error) {
	switch name {
	case "claude":
		return installClaude(code)
	case "cursor":
		return installCursor(dir, code)
	default:
		return "", fmt.Errorf("unknown agent %q: use claude or cursor", name)
	}
}

func installClaude(code string) (string, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return "", fmt.Errorf("the claude CLI is not on PATH; install Claude Code first")
	}
	for _, args := range [][]string{
		{"plugin", "marketplace", "add", marketplace},
		{"plugin", "install", plugin, "--scope", "user"},
	} {
		cmd := exec.Command("claude", args...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("claude %v: %w", args, err)
		}
	}

	return "Restart Claude Code in your project and run: /pushwoosh:setup " + code, nil
}

// installCursor adds the MCP server and the setup rule to the project; Cursor has no plugin CLI.
func installCursor(dir, code string) (string, error) {
	mcpPath := filepath.Join(dir, ".cursor", "mcp.json")
	cfg := map[string]any{}
	if data, err := os.ReadFile(mcpPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return "", fmt.Errorf("%s is not valid JSON: %w", mcpPath, err)
		}
	}
	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["pushwoosh"] = map[string]any{"url": MCPURL}
	cfg["mcpServers"] = servers

	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.MkdirAll(filepath.Dir(mcpPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(mcpPath, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	rule := filepath.Join(dir, ".cursor", "rules", "pushwoosh.mdc")
	if err := os.MkdirAll(filepath.Dir(rule), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(rule, []byte(cursorRule), 0o644); err != nil {
		return "", err
	}

	return "Open the project in Cursor, enable the pushwoosh MCP server and ask: integrate Pushwoosh with install code " + code, nil
}

const cursorRule = `---
description: Integrate the Pushwoosh SDK into this project
alwaysApply: false
---
When asked to integrate Pushwoosh with an install code:
1. Show the plan: ` + "`" + installCmd + " --dry-run <code>`" + `. It handles Next.js, Nuxt, Vite and static sites.
2. After the user agrees, apply it: ` + "`" + installCmd + " --yes <code>`" + `.
3. If it reports an unsupported stack, get the credentials with ` + "`" + installCmd + " redeem <code>`" + `
   and add the Pushwoosh SDK by hand following https://docs.pushwoosh.com, passing both applicationCode
   and apiToken to the SDK init: without the device API token new accounts reject device registration.
`

// installCmd re-downloads the verified binary each time, so nothing stays installed on the machine.
const installCmd = "curl -fsSL https://github.com/Pushwoosh/SDK-Integrator/releases/latest/download/install.sh | sh -s --"
