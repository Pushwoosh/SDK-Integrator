package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCursorKeepsOtherServers(t *testing.T) {
	dir := t.TempDir()
	mcp := filepath.Join(dir, ".cursor", "mcp.json")
	_ = os.MkdirAll(filepath.Dir(mcp), 0o755)
	_ = os.WriteFile(mcp, []byte(`{"mcpServers":{"other":{"url":"https://other.example/mcp"}},"extra":true}`), 0o644)

	next, err := Install("cursor", dir, "pwi_x")
	if err != nil || !strings.Contains(next, "pwi_x") {
		t.Fatalf("next %q, err %v", next, err)
	}

	var cfg struct {
		MCPServers map[string]map[string]string `json:"mcpServers"`
		Extra      bool                         `json:"extra"`
	}
	data, _ := os.ReadFile(mcp)
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.MCPServers["other"]["url"] != "https://other.example/mcp" || cfg.MCPServers["pushwoosh"]["url"] != MCPURL || !cfg.Extra {
		t.Fatalf("mcp.json rewritten wrong: %s", data)
	}
	rule, err := os.ReadFile(filepath.Join(dir, ".cursor", "rules", "pushwoosh.mdc"))
	if err != nil || !strings.Contains(string(rule), installCmd) {
		t.Fatalf("rule: %v", err)
	}
}

func TestCursorRefusesBrokenConfig(t *testing.T) {
	dir := t.TempDir()
	mcp := filepath.Join(dir, ".cursor", "mcp.json")
	_ = os.MkdirAll(filepath.Dir(mcp), 0o755)
	_ = os.WriteFile(mcp, []byte(`{broken`), 0o644)

	if _, err := Install("cursor", dir, "pwi_x"); err == nil {
		t.Fatal("overwrote an invalid mcp.json")
	}
	if data, _ := os.ReadFile(mcp); string(data) != `{broken` {
		t.Fatalf("invalid mcp.json was modified: %s", data)
	}
}

func TestUnknownAgent(t *testing.T) {
	if _, err := Install("vim", t.TempDir(), "pwi_x"); err == nil {
		t.Fatal("unknown agent accepted")
	}
}
