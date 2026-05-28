package adapters

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adamroman0/cosyra-context/internal/cosyra"
)

const (
	claudeSettingsRel = ".claude/settings.local.json"
	codexConfigRel    = ".codex/config.toml"
	opencodePluginRel = ".config/opencode/plugins"
)

func Install(projectRoot string, tools []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	hookDir := filepath.Join(projectRoot, cosyra.DirName, "hooks")
	if err := os.MkdirAll(hookDir, 0o700); err != nil {
		return err
	}
	for _, tool := range tools {
		switch tool {
		case "claude":
			if err := installClaude(projectRoot, hookDir, exe); err != nil {
				return err
			}
		case "codex":
			if err := installCodex(projectRoot, hookDir, exe); err != nil {
				return err
			}
		case "opencode":
			if err := installOpenCode(projectRoot, exe); err != nil {
				return err
			}
		}
	}
	return nil
}

func Uninstall(projectRoot string, tools []string) error {
	for _, tool := range tools {
		switch tool {
		case "claude":
			if err := uninstallClaude(projectRoot); err != nil {
				return err
			}
		case "codex":
			if err := uninstallCodex(projectRoot); err != nil {
				return err
			}
		case "opencode":
			if err := uninstallOpenCode(projectRoot); err != nil {
				return err
			}
		}
	}
	return nil
}

func installClaude(projectRoot string, hookDir string, exe string) error {
	scriptPath := filepath.Join(hookDir, "claude-event")
	if err := writeHookScript(scriptPath, exe, projectRoot, "claude"); err != nil {
		return err
	}
	settingsPath := filepath.Join(projectRoot, claudeSettingsRel)
	return mergeJSONFile(settingsPath, 0o600, func(cfg map[string]any) (bool, error) {
		hooks, _ := cfg["hooks"].(map[string]any)
		if hooks == nil {
			hooks = map[string]any{}
		}
		changed := false
		for _, event := range []string{"SessionStart", "Stop"} {
			existing, _ := hooks[event].([]any)
			if !containsCommandHook(existing, scriptPath) {
				existing = append(existing, map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{"type": "command", "command": scriptPath},
					},
				})
				hooks[event] = existing
				changed = true
			}
		}
		if changed {
			cfg["hooks"] = hooks
		}
		return changed, nil
	})
}

func uninstallClaude(projectRoot string) error {
	settingsPath := filepath.Join(projectRoot, claudeSettingsRel)
	hookPrefix := filepath.Join(projectRoot, cosyra.DirName, "hooks", "claude-event")
	return mergeJSONFile(settingsPath, 0o600, func(cfg map[string]any) (bool, error) {
		hooks, _ := cfg["hooks"].(map[string]any)
		if hooks == nil {
			return false, nil
		}
		changed := false
		for _, event := range []string{"SessionStart", "Stop"} {
			existing, _ := hooks[event].([]any)
			filtered := removeCommandHook(existing, hookPrefix)
			if len(filtered) != len(existing) {
				hooks[event] = filtered
				changed = true
			}
		}
		return changed, nil
	})
}

func installCodex(projectRoot string, hookDir string, exe string) error {
	scriptPath := filepath.Join(hookDir, "codex-event")
	if err := writeHookScript(scriptPath, exe, projectRoot, "codex"); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, codexConfigRel)
	block := fmt.Sprintf(`# cosyra-context:start project=%s
[[hooks.SessionStart]]
matcher = ""
[[hooks.SessionStart.hooks]]
type = "command"
command = %q

[[hooks.Stop]]
matcher = ""
[[hooks.Stop.hooks]]
type = "command"
command = %q
# cosyra-context:end project=%s
`, projectRoot, scriptPath, scriptPath, projectRoot)
	return replaceMarkedBlock(configPath, projectRoot, block)
}

func uninstallCodex(projectRoot string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, codexConfigRel)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := removeMarkedBlock(string(data), projectRoot)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(configPath, []byte(content), 0o600)
}

func installOpenCode(projectRoot string, exe string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	pluginDir := filepath.Join(home, opencodePluginRel)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return err
	}
	pluginPath := filepath.Join(pluginDir, "cosyra-context-"+shortHash(projectRoot)+".js")
	src := fmt.Sprintf(`// Cosyra Context OpenCode adapter. Generated; safe to remove with "cosyra off".
const child_process = require("child_process");
const PROJECT_ROOT = %q;
const COSYRA = %q;

module.exports = async function ({ event }) {
  try {
    const cwd = process.cwd();
    if (cwd !== PROJECT_ROOT && !cwd.startsWith(PROJECT_ROOT + "/")) return;
    const name = event && event.type ? String(event.type) : "unknown";
    if (name !== "session.idle" && name !== "permission.asked") return;
    const hook = name === "session.idle" ? "stop" : "stop";
    const payload = JSON.stringify({ hook_event_name: name, session_id: event && event.sessionID || "" });
    child_process.spawnSync(COSYRA, ["hook", hook, "--agent", "opencode", "--project", PROJECT_ROOT], {
      input: payload,
      stdio: ["pipe", "ignore", "ignore"],
      timeout: 2000
    });
  } catch (_) {}
};
`, projectRoot, exe)
	return os.WriteFile(pluginPath, []byte(src), 0o644)
}

func uninstallOpenCode(projectRoot string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, opencodePluginRel, "cosyra-context-"+shortHash(projectRoot)+".js")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeHookScript(path string, exe string, projectRoot string, agent string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	eventCase := `hook_cmd="stop"`
	if agent == "claude" || agent == "codex" {
		eventCase = `hook_cmd="stop"
case "$hook_event" in
  SessionStart)
    hook_cmd="start"
    ;;
esac`
	}
	script := fmt.Sprintf(`#!/bin/sh
input=$(cat)
hook_event=$(printf '%%s' "$input" | grep -oE '"hook_event_name"[[:space:]]*:[[:space:]]*"[A-Za-z_]+"' | head -1 | sed -E 's/.*"([A-Za-z_]+)"$/\1/')
[ -n "$hook_event" ] || hook_event="unknown"
%s
printf '%%s' "$input" | %q hook "$hook_cmd" --agent %q --project %q >/dev/null 2>&1 || true
`, eventCase, exe, agent, projectRoot)
	return os.WriteFile(path, []byte(script), 0o700)
}

func mergeJSONFile(path string, mode os.FileMode, mutate func(map[string]any) (bool, error)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cfg := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	changed, err := mutate(cfg)
	if err != nil || !changed {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, mode)
}

func containsCommandHook(entries []any, command string) bool {
	for _, entry := range entries {
		entryMap, _ := entry.(map[string]any)
		hooksList, _ := entryMap["hooks"].([]any)
		for _, h := range hooksList {
			hookMap, _ := h.(map[string]any)
			if hookMap["command"] == command {
				return true
			}
		}
	}
	return false
}

func removeCommandHook(entries []any, command string) []any {
	var out []any
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			out = append(out, entry)
			continue
		}
		hooksList, _ := entryMap["hooks"].([]any)
		var keptHooks []any
		for _, h := range hooksList {
			hookMap, _ := h.(map[string]any)
			if hookMap["command"] == command {
				continue
			}
			keptHooks = append(keptHooks, h)
		}
		if len(keptHooks) == 0 {
			continue
		}
		entryMap["hooks"] = keptHooks
		out = append(out, entryMap)
	}
	return out
}

func replaceMarkedBlock(path string, projectRoot string, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := removeMarkedBlock(string(data), projectRoot)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += block
	return os.WriteFile(path, []byte(content), 0o600)
}

func removeMarkedBlock(content string, projectRoot string) string {
	start := "# cosyra-context:start project=" + projectRoot
	end := "# cosyra-context:end project=" + projectRoot
	lines := strings.Split(content, "\n")
	var kept []string
	skip := false
	for _, line := range lines {
		if line == start {
			skip = true
			continue
		}
		if line == end {
			skip = false
			continue
		}
		if !skip {
			kept = append(kept, line)
		}
	}
	return strings.TrimRight(strings.Join(kept, "\n"), "\n")
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}
