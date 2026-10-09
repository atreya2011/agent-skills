package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const window = 6 * time.Hour

func TestHerdrSources(t *testing.T) {
	w := newWorld(t, "")
	base := w.config(t)
	r := runner{cmds: base.Commands}
	ws, err := r.herdrList(t.Context(), "workspace")
	require.NoError(t, err)
	tabs, err := r.herdrList(t.Context(), "tab")
	require.NoError(t, err)
	agents, err := r.herdrList(t.Context(), "agent")
	require.NoError(t, err)

	tests := []struct {
		name           string
		pinned         Pinned
		wantWorkspaces []Workspace
		wantTabs       []Tab
	}{
		{"the fixture lists join into workspaces and tabs", base.Pinned,
			[]Workspace{
				{ID: "w1", Label: "alpha", State: "idle", OrchestratorTab: "w1:t1"},
				{ID: "w2", Label: "notes", State: "blocked"},
				{ID: "w3", Label: "beta", State: "working"},
			},
			[]Tab{
				{ID: "w1:t1", Label: "orchestrator", Workspace: "w1", Agent: "claude", Target: "alpha-orch", State: "idle", CWD: filepath.Join(w.Root, "alpha-wt"), SessionID: sessAlpha},
				{ID: "w1:t2", Label: "shell", Workspace: "w1", State: "unknown"},
				{ID: "w2:t1", Label: "keep-me", Workspace: "w2", Agent: "claude", Target: "w2:p1", State: "blocked", Pinned: true, CWD: filepath.Join(w.Root, "notes"), SessionID: sessNotes},
				{ID: "w3:t1", Label: "impl", Workspace: "w3", Agent: "codex", Target: "beta-impl", State: "working", CWD: filepath.Join(w.Root, "beta")},
				{ID: "w3:t2", Label: "helper", Workspace: "w3", Agent: "codex", Target: "w3:p2", State: "done", CWD: filepath.Join(w.Root, "beta")},
			}},
		{"a pinned workspace label pins the workspace and every tab in it", Pinned{Workspaces: []string{"beta"}},
			[]Workspace{
				{ID: "w1", Label: "alpha", State: "idle", OrchestratorTab: "w1:t1"},
				{ID: "w2", Label: "notes", State: "blocked"},
				{ID: "w3", Label: "beta", State: "working", Pinned: true},
			},
			[]Tab{
				{ID: "w1:t1", Label: "orchestrator", Workspace: "w1", Agent: "claude", Target: "alpha-orch", State: "idle", CWD: filepath.Join(w.Root, "alpha-wt"), SessionID: sessAlpha},
				{ID: "w1:t2", Label: "shell", Workspace: "w1", State: "unknown"},
				{ID: "w2:t1", Label: "keep-me", Workspace: "w2", Agent: "claude", Target: "w2:p1", State: "blocked", CWD: filepath.Join(w.Root, "notes"), SessionID: sessNotes},
				{ID: "w3:t1", Label: "impl", Workspace: "w3", Agent: "codex", Target: "beta-impl", State: "working", Pinned: true, CWD: filepath.Join(w.Root, "beta")},
				{ID: "w3:t2", Label: "helper", Workspace: "w3", Agent: "codex", Target: "w3:p2", State: "done", Pinned: true, CWD: filepath.Join(w.Root, "beta")},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.Pinned = tt.pinned
			workspaces, outTabs := buildHerdr(cfg, ws.Result.Workspaces, tabs.Result.Tabs, agents.Result.Agents)
			assert.Equal(t, tt.wantWorkspaces, workspaces)
			assert.Equal(t, tt.wantTabs, outTabs)
		})
	}
}

func TestCommandFailures(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, w *world, cmds *Commands)
		wantErr string
	}{
		{"no binary", func(_ *testing.T, w *world, c *Commands) { c.Herdr = filepath.Join(w.Root, "absent") }, "no such file"},
		{"no recording", func(t *testing.T, w *world, _ *Commands) {
			require.NoError(t, os.Remove(filepath.Join(w.Fixtures, "herdr", "workspace_list.out")))
		}, "no recording for herdr workspace_list"},
		{"nonzero exit with a message", func(t *testing.T, w *world, _ *Commands) {
			writeFile(t, filepath.Join(w.Fixtures, "herdr", "workspace_list.code"), "1\n", time.Time{})
			writeFile(t, filepath.Join(w.Fixtures, "herdr", "workspace_list.err"), "server is not running\nsecond line\n", time.Time{})
		}, "server is not running"},
		{"output that is not JSON", func(t *testing.T, w *world, _ *Commands) {
			writeFile(t, filepath.Join(w.Fixtures, "herdr", "workspace_list.out"), "<html>", time.Time{})
		}, "unreadable output"},
		{"the run deadline has passed", func(*testing.T, *world, *Commands) {}, "the run deadline passed"},
		{"the run was cancelled", func(*testing.T, *world, *Commands) {}, "the run was cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t, "")
			cmds := w.config(t).Commands
			tt.prepare(t, w, &cmds)
			ctx := t.Context()
			switch tt.wantErr {
			case "the run deadline passed":
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer cancel()
			case "the run was cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			_, err := runner{cmds: cmds}.herdrList(ctx, "workspace")
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestTodos(t *testing.T) {
	w := newWorld(t, "")
	r := runner{cmds: w.config(t).Commands}
	todos, err := r.todos(t.Context(), testNow, window)
	require.NoError(t, err)

	byDesc := map[string]Todo{}
	for _, td := range todos {
		byDesc[td.Description] = td
	}
	assert.Len(t, todos, 12, "open todos plus the one completed inside the window; deleted and old completed are dropped")
	assert.NotContains(t, byDesc, "Ancient finished chore")
	assert.NotContains(t, byDesc, "Deleted item")

	assert.Equal(t, "example/alpha", byDesc["Add the parser"].slug())
	assert.Equal(t, "example/gamma", byDesc["Check the gamma bug"].slug())
	assert.Empty(t, byDesc["Tune the retry limits"].slug())

	tests := []struct {
		desc string
		want Todo
	}{
		{"Add the parser", Todo{Domain: "work", Assignee: "cto", Status: "pending",
			Annotations: []string{"pr: https://github.com/example/alpha/pull/7"},
			Links:       []Link{{Kind: "pr", URL: "https://github.com/example/alpha/pull/7"}}}},
		{"Book the dentist", Todo{Domain: "personal", Assignee: "cos", Status: "pending", Annotations: []string{}, Links: []Link{}}},
		{"Fix the beta flake", Todo{Assignee: "cto", Status: "pending", Annotations: []string{"https://github.com/example/beta/issues/9"},
			Links: []Link{{Kind: "issue", URL: "https://github.com/example/beta/issues/9"}}}},
		{"Tune the retry limits", Todo{Assignee: "cto", Status: "pending", SessionID: sessOldUnbound, Dispatched: true,
			Annotations: []string{
				"session: " + filepath.Join(w.Root, "claude-main", "projects", "-work-alpha", sessOldUnbound+".jsonl"),
				"run: beta-2"},
			Links: []Link{}}},
		{"Old finished chore", Todo{Assignee: "cto", Status: "completed", ClosedAt: "2026-10-08T10:00:00Z", Annotations: []string{}, Links: []Link{}}},
		{"Land the merged change", Todo{Assignee: "cto", Status: "pending",
			Annotations: []string{"https://github.com/example/alpha/issues/4 commit: " + w.ShaMerged},
			Links: []Link{{Kind: "issue", URL: "https://github.com/example/alpha/issues/4"},
				{Kind: "commit", SHA: w.ShaMerged, State: "unchecked"}}}},
		{"Land the other merged change", Todo{Assignee: "cto", Status: "pending",
			Annotations: []string{"commit: " + w.ShaMerged, "https://github.com/example/alpha/issues/6"},
			Links: []Link{{Kind: "commit", SHA: w.ShaMerged, State: "unchecked"},
				{Kind: "issue", URL: "https://github.com/example/alpha/issues/6"}}}},
		{"Check the parser question", Todo{Assignee: "cto", Ask: true, Status: "pending", Links: []Link{},
			Annotations: []string{"ask: Is the parser work done? See https://github.com/example/alpha/pull/7 | suggested: done | cto todo_state done 0.70 escalate 1a2b3c4d 2026-10-03"}}},
		{"Plan the offsite", Todo{Status: "pending", Annotations: []string{}, Links: []Link{}}},
		{"Check the gamma bug", Todo{Assignee: "cto", Ask: true, Status: "pending",
			Annotations: []string{"https://github.com/example/gamma/issues/1"},
			Links:       []Link{{Kind: "issue", URL: "https://github.com/example/gamma/issues/1"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := byDesc[tt.desc]
			// The test compares the exported fields; identity and link slugs are checked below.
			for i := range got.Links {
				got.Links[i].slug = ""
			}
			got.UUID, got.Description = "", ""
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGitFacts(t *testing.T) {
	w := newWorld(t, "")
	r := runner{cmds: w.config(t).Commands}
	alpha := filepath.Join(w.Root, "alpha")

	roots := []struct {
		name     string
		cwd      string
		wantRoot string
		wantRepo bool
	}{
		{"a linked worktree folds into its main checkout", filepath.Join(w.Root, "alpha-wt"), alpha, true},
		{"the main checkout", alpha, alpha, true},
		{"a directory outside any repository", filepath.Join(w.Root, "notes"), "", false},
		{"a directory that is gone", filepath.Join(w.Root, "removed"), "", false},
	}
	for _, tt := range roots {
		t.Run(tt.name, func(t *testing.T) {
			root, isRepo, err := r.gitRoot(t.Context(), tt.cwd)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRoot, root)
			assert.Equal(t, tt.wantRepo, isRepo)
		})
	}

	t.Run("details of a checkout with an origin", func(t *testing.T) {
		d, err := r.gitDetails(t.Context(), alpha)
		require.NoError(t, err)
		assert.Equal(t, "example/alpha", d.Slug)
		assert.Equal(t, "feat/x", d.Branch)
		assert.Equal(t, "origin/main", d.DefaultBranch)
		assert.Equal(t, 1, d.Dirty, "one untracked file")
	})

	t.Run("details of a checkout with an ssh origin and no origin HEAD", func(t *testing.T) {
		d, err := r.gitDetails(t.Context(), filepath.Join(w.Root, "beta"))
		require.NoError(t, err)
		assert.Equal(t, "example/beta", d.Slug)
		assert.Empty(t, d.DefaultBranch)
	})

	t.Run("a path that is not a repository fails", func(t *testing.T) {
		_, err := r.gitDetails(t.Context(), filepath.Join(w.Root, "notes"))
		require.Error(t, err)
	})

	for _, tt := range []struct {
		name, def, sha, want string
	}{
		{"a commit on the default branch", "origin/main", w.ShaMerged, "on_default"},
		{"a commit only on a feature branch", "origin/main", w.ShaOpen, "not_on_default"},
		{"a commit the repository does not know", "origin/main", "deadbeefdeadbeef", "unchecked"},
		{"a repository without a default branch", "", w.ShaMerged, "unchecked"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.onDefault(t.Context(), alpha, tt.def, tt.sha))
		})
	}

	t.Run("origin URL forms", func(t *testing.T) {
		for url, want := range map[string]string{
			"https://github.com/example/alpha.git": "example/alpha",
			"https://github.com/example/alpha":     "example/alpha",
			"ssh://github.com/example/alpha.git":   "example/alpha",
			"https://gitlab.example/example/alpha": "",
		} {
			m := originSlug.FindStringSubmatch(url)
			got := ""
			if m != nil {
				got = m[1]
			}
			assert.Equal(t, want, got, url)
		}
	})
}

func TestGitHubFacts(t *testing.T) {
	w := newWorld(t, "")
	r := runner{cmds: w.config(t).Commands}

	prs, err := r.pullRequests(t.Context(), "example/alpha")
	require.NoError(t, err)
	assert.Equal(t, []PR{
		{Number: 7, Title: "Add the parser", State: "MERGED", Branch: "feat/parser", MergedAt: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC), URL: "https://github.com/example/alpha/pull/7"},
		{Number: 8, Title: "Draft the docs", State: "OPEN", Branch: "feat/y", Draft: true, URL: "https://github.com/example/alpha/pull/8"},
	}, prs)
	_, err = r.pullRequests(t.Context(), "example/missing")
	require.ErrorContains(t, err, "no recording")

	tests := []struct {
		name    string
		link    Link
		want    Link
		wantErr string
	}{
		{"a merged pull request", Link{Kind: "pr", URL: "https://github.com/example/alpha/pull/7"},
			Link{Kind: "pr", URL: "https://github.com/example/alpha/pull/7", State: "MERGED"}, ""},
		{"an open issue with a label", Link{Kind: "issue", URL: "https://github.com/example/alpha/issues/3"},
			Link{Kind: "issue", URL: "https://github.com/example/alpha/issues/3", State: "OPEN", Labels: []string{"ready-for-agent"}}, ""},
		{"an issue closed as completed", Link{Kind: "issue", URL: "https://github.com/example/beta/issues/9"},
			Link{Kind: "issue", URL: "https://github.com/example/beta/issues/9", State: "CLOSED", StateReason: "COMPLETED"}, ""},
		{"an issue that cannot be read", Link{Kind: "issue", URL: "https://github.com/example/gamma/issues/1"},
			Link{Kind: "issue", URL: "https://github.com/example/gamma/issues/1"}, "no recording"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.link
			err := r.fillLink(t.Context(), &l)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, l)
		})
	}
}

func TestTranscriptIndex(t *testing.T) {
	w := newWorld(t, "")
	cfg := w.config(t)
	ix, bad := buildIndex(cfg.Profiles, testNow, window)
	assert.Empty(t, bad)

	assert.Equal(t, []ProfileInfo{
		{Name: "main", Kind: "claude", LastActivity: testNow.Add(-10 * time.Minute), SessionsInWindow: 3},
		{Name: "alias", Kind: "claude", SameAs: "main"},
		{Name: "codex", Kind: "codex", LastActivity: testNow.Add(-20 * time.Minute), SessionsInWindow: 1},
	}, ix.infos, "the symlinked profile shares the first profile's sessions")

	assert.Len(t, ix.byID, 5, "four claude sessions and one codex session; subagent files are not sessions")
	assert.Equal(t, "main", ix.byID[sessAlpha].Profile)
	assert.Equal(t, "codex", ix.byID[sessBeta].Profile)
	assert.NotContains(t, ix.byID, "agent-1")
	assert.Len(t, ix.recent(testNow, window), 4, "S5 was written 30 hours ago")

	t.Run("a missing directory is unreadable and the others still load", func(t *testing.T) {
		profiles := append([]Profile{{Name: "gone", Kind: "claude", Dir: filepath.Join(w.Root, "absent")}}, cfg.Profiles...)
		ix, bad := buildIndex(profiles, testNow, window)
		require.Len(t, bad, 1)
		assert.Equal(t, "gone", bad[0].Target)
		assert.Len(t, ix.byID, 5)
	})
}

func TestReadTranscript(t *testing.T) {
	w := newWorld(t, "")
	ix, _ := buildIndex(w.config(t).Profiles, testNow, window)
	tests := []struct {
		name        string
		id          string
		tailBytes   int
		chars       int
		wantExcerpt string
		wantCWD     string
	}{
		{"claude text without tool results", sessAlpha, 65536, 400,
			"user: please add the parser\nassistant: Writing the parser now.\nassistant: Parser added and tests pass.",
			filepath.Join(w.Root, "alpha-wt")},
		{"codex text without developer messages or calls", sessBeta, 65536, 400,
			"user: run the tests\nassistant: Tests pass.", filepath.Join(w.Root, "beta")},
		{"the excerpt keeps the last characters", sessAlpha, 65536, 20, "ser: Parser added and tests pass."[len("ser: Parser added and tests pass.")-20:], filepath.Join(w.Root, "alpha-wt")},
		{"a tail that cuts the first line drops it", sessAlpha, 330, 400,
			"assistant: Parser added and tests pass.", filepath.Join(w.Root, "alpha-wt")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excerpt, cwd, err := readTranscript(ix.byID[tt.id], tt.tailBytes, tt.chars)
			require.NoError(t, err)
			assert.Equal(t, tt.wantExcerpt, excerpt)
			assert.Equal(t, tt.wantCWD, cwd)
		})
	}

	t.Run("lines that are not JSON are skipped", func(t *testing.T) {
		path := filepath.Join(w.Root, "garbage.jsonl")
		writeFile(t, path, "not json\n\x00\x01\n"+`{"type":"assistant","message":{"content":"ok"}}`+"\n{\"type\":", time.Time{})
		excerpt, _, err := readTranscript(transcriptFile{Kind: "claude", Path: path}, 4096, 100)
		require.NoError(t, err)
		assert.Equal(t, "assistant: ok", excerpt)
	})
	t.Run("a missing file is an error", func(t *testing.T) {
		_, _, err := readTranscript(transcriptFile{Kind: "claude", Path: filepath.Join(w.Root, "absent.jsonl")}, 10, 10)
		require.Error(t, err)
	})
}
