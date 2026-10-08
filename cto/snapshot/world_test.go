package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Session IDs of the synthetic world.
const (
	sessAlpha      = "11111111-1111-4111-8111-111111111111" // claude, bound to tab w1:t1
	sessNotes      = "22222222-2222-4222-8222-222222222222" // claude, bound to tab w2:t1
	sessBeta       = "33333333-3333-4333-8333-333333333333" // codex; herdr gives codex agents no session ID, so no tab is bound
	sessUnbound    = "44444444-4444-4444-8444-444444444444" // claude, no tab, in the window
	sessOldUnbound = "55555555-5555-4555-8555-555555555555" // claude, no tab, outside the window
)

// world is a synthetic machine: real git repositories, transcripts for two
// profiles, a vault, synthetic herdr, gh and task output, and a local file that
// points at all of it.
type world struct {
	Root      string
	Local     string
	Fixtures  string
	ShaMerged string
	ShaOpen   string
}

func runIn(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s %v: %s", name, args, out)
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, path, content string, mtime time.Time) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	if !mtime.IsZero() {
		require.NoError(t, os.Chtimes(path, mtime, mtime))
	}
}

// newWorld builds the world under a temporary directory and points the process
// at its stand-in programs. gateURL goes into the local file.
func newWorld(t *testing.T, gateURL string) *world {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	w := &world{Root: root, Local: filepath.Join(root, "cto.md"), Fixtures: filepath.Join(root, "fixtures")}

	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": os.DevNull, "GIT_CONFIG_SYSTEM": os.DevNull,
		"GIT_AUTHOR_NAME": "Tester", "GIT_AUTHOR_EMAIL": "tester",
		"GIT_COMMITTER_NAME": "Tester", "GIT_COMMITTER_EMAIL": "tester",
		"GIT_AUTHOR_DATE": "2026-10-07T10:00:00Z", "GIT_COMMITTER_DATE": "2026-10-07T10:00:00Z",
	} {
		t.Setenv(k, v)
	}
	alpha := filepath.Join(root, "alpha")
	require.NoError(t, os.MkdirAll(alpha, 0o755))
	runIn(t, alpha, "git", "init", "-q", "-b", "main")
	writeFile(t, filepath.Join(alpha, "a.txt"), "one\n", time.Time{})
	runIn(t, alpha, "git", "add", "a.txt")
	runIn(t, alpha, "git", "commit", "-q", "-m", "add the skeleton")
	writeFile(t, filepath.Join(alpha, "a.txt"), "two\n", time.Time{})
	runIn(t, alpha, "git", "commit", "-q", "-a", "-m", "add the parser")
	w.ShaMerged = runIn(t, alpha, "git", "rev-parse", "HEAD")
	runIn(t, alpha, "git", "remote", "add", "origin", "https://github.com/example/alpha.git")
	runIn(t, alpha, "git", "update-ref", "refs/remotes/origin/main", "HEAD")
	runIn(t, alpha, "git", "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	runIn(t, alpha, "git", "checkout", "-q", "-b", "feat/x")
	writeFile(t, filepath.Join(alpha, "b.txt"), "three\n", time.Time{})
	runIn(t, alpha, "git", "add", "b.txt")
	runIn(t, alpha, "git", "commit", "-q", "-m", "start the docs")
	w.ShaOpen = runIn(t, alpha, "git", "rev-parse", "HEAD")
	writeFile(t, filepath.Join(alpha, "scratch.txt"), "untracked\n", time.Time{})
	runIn(t, alpha, "git", "worktree", "add", "-q", "-b", "feat/y", filepath.Join(root, "alpha-wt"), "main")

	beta := filepath.Join(root, "beta")
	require.NoError(t, os.MkdirAll(beta, 0o755))
	runIn(t, beta, "git", "init", "-q", "-b", "main")
	writeFile(t, filepath.Join(beta, "r.txt"), "r\n", time.Time{})
	runIn(t, beta, "git", "add", "r.txt")
	runIn(t, beta, "git", "commit", "-q", "-m", "first")
	runIn(t, beta, "git", "remote", "add", "origin", "ssh://github.com/example/beta.git")
	require.NoError(t, os.MkdirAll(filepath.Join(root, "notes"), 0o755))

	now := testNow
	claude := filepath.Join(root, "claude-main", "projects")
	line := func(typ, cwd, content string) string {
		return fmt.Sprintf(`{"type":%q,"cwd":%q,"message":{"role":%q,"content":%s}}`, typ, cwd, typ, content) + "\n"
	}
	writeFile(t, filepath.Join(claude, "-work-alpha", sessAlpha+".jsonl"),
		`{"type":"permission-mode","sessionId":"x"}`+"\n"+
			line("user", filepath.Join(root, "alpha-wt"), `"please add the parser"`)+
			line("assistant", filepath.Join(root, "alpha-wt"), `[{"type":"text","text":"Writing the parser now."},{"type":"tool_use","name":"Edit","input":{}}]`)+
			line("user", filepath.Join(root, "alpha-wt"), `[{"type":"tool_result","content":"ok"}]`)+
			line("assistant", filepath.Join(root, "alpha-wt"), `[{"type":"text","text":"Parser added and tests pass."}]`),
		now.Add(-10*time.Minute))
	writeFile(t, filepath.Join(claude, "-work-alpha", sessAlpha, "subagents", "agent-1.jsonl"), line("user", alpha, `"subagent noise"`), now)
	writeFile(t, filepath.Join(claude, "-work-notes", sessNotes+".jsonl"),
		line("user", filepath.Join(root, "notes"), `"draft the note"`)+
			line("assistant", filepath.Join(root, "notes"), `[{"type":"text","text":"May I overwrite notes.md?"}]`),
		now.Add(-3*time.Hour))
	writeFile(t, filepath.Join(claude, "-work-alpha", sessUnbound+".jsonl"),
		line("assistant", alpha, `[{"type":"text","text":"Reviewing the diff for PR 8."}]`), now.Add(-time.Hour))
	writeFile(t, filepath.Join(claude, "-work-alpha", sessOldUnbound+".jsonl"),
		line("user", alpha, `"tune the retry limits"`)+line("assistant", alpha, `[{"type":"text","text":"Limits tuned to 5 attempts."}]`),
		now.Add(-30*time.Hour))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "claude-alias"), 0o755))
	require.NoError(t, os.Symlink(claude, filepath.Join(root, "claude-alias", "projects")))
	codexLine := func(payload string) string { return `{"timestamp":"2026-10-08T11:40:00Z",` + payload + "}\n" }
	writeFile(t, filepath.Join(root, "codex-main", "sessions", "2026", "10", "08", "rollout-2026-10-08T11-40-00-"+sessBeta+".jsonl"),
		codexLine(fmt.Sprintf(`"type":"session_meta","payload":{"id":%q,"cwd":%q}`, sessBeta, beta))+
			codexLine(fmt.Sprintf(`"type":"turn_context","payload":{"cwd":%q}`, beta))+
			codexLine(`"type":"response_item","payload":{"type":"message","role":"developer","content":[{"type":"input_text","text":"hidden rules"}]}`)+
			codexLine(`"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"run the tests"}]}`)+
			codexLine(`"type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{}"}`)+
			codexLine(`"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Tests pass."}]}`),
		now.Add(-20*time.Minute))

	writeFile(t, filepath.Join(root, "vaults", "alpha", "index.md"), "# index\n", now.Add(-2*time.Hour))

	subs := strings.NewReplacer("{{root}}", root, "{{sha_merged}}", w.ShaMerged, "{{sha_unmerged}}", w.ShaOpen)
	require.NoError(t, filepath.WalkDir("testdata/base/fixtures", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("testdata/base/fixtures", path)
		writeFile(t, filepath.Join(w.Fixtures, rel), subs.Replace(string(data)), time.Time{})
		return nil
	}))
	bin := filepath.Join(root, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	self, err := os.Executable()
	require.NoError(t, err)
	for _, name := range shimNames {
		require.NoError(t, os.Symlink(self, filepath.Join(bin, name)))
	}
	t.Setenv("SHIM_FIXTURES", w.Fixtures)

	gate := ""
	if gateURL != "" {
		gate = fmt.Sprintf("url = %q\n", gateURL)
	}
	local := subs.Replace(fmt.Sprintf("# CTO local file\n\nSynthetic.\n\n```toml\n"+
		"[commands]\ntask = %q\nherdr = %q\ngh = %q\n\n"+
		"[pinned]\ntabs = [\"keep-me\"]\nworkspaces = []\n\n"+
		"[vaults]\n\"example/alpha\" = \"{{root}}/vaults/alpha\"\n\n"+
		"[[profiles]]\nname = \"main\"\nkind = \"claude\"\ndir = \"{{root}}/claude-main/projects\"\n\n"+
		"[[profiles]]\nname = \"alias\"\nkind = \"claude\"\ndir = \"{{root}}/claude-alias/projects\"\n\n"+
		"[[profiles]]\nname = \"codex\"\nkind = \"codex\"\ndir = \"{{root}}/codex-main/sessions\"\n\n"+
		"[thresholds]\nfloor = 0.6\nlow = 0.6\nhigh = 0.8\nwindow_hours = 6\ntail_bytes = 65536\nexcerpt_chars = 400\n\n"+
		"[gate]\n%slog = \"{{root}}/state/gate.log\"\n```\n",
		filepath.Join(bin, "task"), filepath.Join(bin, "herdr"), filepath.Join(bin, "gh"), gate))
	writeFile(t, w.Local, local, time.Time{})
	return w
}

// config loads the world's local file.
func (w *world) config(t *testing.T) Config {
	t.Helper()
	cfg, err := loadConfig(w.Local)
	require.NoError(t, err)
	return cfg
}
