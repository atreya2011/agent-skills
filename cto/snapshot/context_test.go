package main

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikiAnnotationNames(t *testing.T) {
	cases := []struct {
		name, annotation string
		want             []string
	}{
		{"bare list", "wiki: build-upgrade-schedule, collaborator | src: work/page.md", []string{"build-upgrade-schedule", "collaborator"}},
		{"explanatory semicolons", "Wiki: duplicate-audit.md — confirmed-events table row 1; excluded verbatim from PR #141/#142; stale ID needs refresh; 'possible coincidence' resolved", []string{"duplicate-audit.md"}},
		{"paths and line locators", "Wiki: ~/vaults/personal/concepts/freeze-tracer.md (Origin section: negotiation, Option B, no escape hatches); concepts/annotation-format.md (spec); index.md:95. Repo: ~/project", []string{"concepts/annotation-format.md", "index.md:95", "~/vaults/personal/concepts/freeze-tracer.md"}},
		{"daily label", "Wiki: ~/vaults/work/projects/article.md; people/author.md; daily sources/worklog/daily.md (open)", []string{"people/author.md", "sources/worklog/daily.md", "~/vaults/work/projects/article.md"}},
		{"non-ASCII path", "wiki: articles/日本語-note.md", []string{"articles/日本語-note.md"}},
		{"spaced filename", "wiki: articles/team plans.md", []string{"articles/team plans.md"}},
		{"spaced directory", "wiki: articles/team plans/日本語-note.md", []string{"articles/team plans/日本語-note.md"}},
		{"spaced page", "wiki: team plans.md", []string{"team plans.md"}},
		{"spaced first directory", "wiki: team plans/article.md", []string{"team plans/article.md"}},
		{"combining mark", "wiki: articles/café.md", []string{"articles/café.md"}},
		{"non-ASCII page", "wiki: 日本語-note.md:12", []string{"日本語-note.md:12"}},
		{"own question excluded", "ask: wiki: old-verdict", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.want, wikiNames([]string{tt.annotation})) })
	}
}

func TestTodoContextFiles(t *testing.T) {
	cases := []struct {
		name, path, reader string
		symlink            bool
	}{}
	for _, name := range []string{"complete context", "checkout evaluation", "facts without judgments", "non-ASCII page", "spaced filename", "missing page", "ambiguous name", "cross-vault symlink", "missing handoff directory"} {
		cases = append(cases, struct {
			name, path, reader string
			symlink            bool
		}{name: name})
	}
	for _, file := range []struct{ name, path string }{
		{"environment", ".env"}, {"private key", "signing.key"}, {"certificate", "signing.pem"},
		{"private metadata", "credentials.md"}, {"private directory", "credentials/page.md"},
	} {
		for _, reader := range []string{"wiki", "handoff"} {
			for _, link := range []bool{false, true} {
				cases = append(cases, struct {
					name, path, reader string
					symlink            bool
				}{
					name: fmt.Sprintf("%s/%s/symlink=%t", file.name, reader, link), path: file.path, reader: reader, symlink: link,
				})
			}
		}
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			kind := tt.name
			root := t.TempDir()
			vault := filepath.Join(root, "vault")
			handoffs := filepath.Join(root, "handoffs")
			foreign := filepath.Join(root, "foreign")
			for _, dir := range []string{filepath.Join(vault, "projects"), handoffs, foreign} {
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			page := filepath.Join(vault, "projects", "project.md")
			text := strings.Repeat("Full evidence. ", 2000)
			require.NoError(t, os.WriteFile(page, []byte(text), 0o600))
			todo := Todo{UUID: "12345678-1234-4321-9876-abcdef123456", Description: "finish work", Domain: "work", Annotations: []string{"wiki: project", "ask: previous verdict"}, Links: []Link{{Kind: "pr", URL: "https://github.com/example/repo/pull/123"}}}
			for name, body := range map[string]string{"full.md": "todo " + todo.UUID, "short.md": "todo " + todo.UUID[:8], "pr.md": todo.Links[0].URL, "other.md": "123456780"} {
				require.NoError(t, os.WriteFile(filepath.Join(handoffs, name), []byte(body), 0o600))
			}
			cfg := Config{Vaults: map[string]string{"work": vault, "other": foreign}, Context: ContextConfig{HandoffDir: handoffs}, OwnerPolicy: "code belongs to +cto"}
			if kind == "checkout evaluation" || kind == "facts without judgments" {
				t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
				t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
				checkout := filepath.Join(root, "checkout")
				require.NoError(t, os.MkdirAll(checkout, 0o700))
				runIn(t, checkout, "git", "init", "-q", "-b", "main")
				runIn(t, checkout, "git", "config", "user.name", "Fixture")
				runIn(t, checkout, "git", "config", "user.email", "fixture")
				writeFile(t, filepath.Join(checkout, "file.txt"), "synthetic content", time.Time{})
				runIn(t, checkout, "git", "add", "file.txt")
				runIn(t, checkout, "git", "commit", "-q", "-m", "fixture")
				sha := runIn(t, checkout, "git", "rev-parse", "HEAD")
				runIn(t, checkout, "git", "remote", "add", "origin", "https://github.com/example/repo.git")
				runIn(t, checkout, "git", "update-ref", "refs/remotes/origin/main", "HEAD")
				runIn(t, checkout, "git", "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
				cfg.Commands = Commands{Git: "git", Gh: filepath.Join(root, "unavailable"), Herdr: filepath.Join(root, "unavailable"), Task: filepath.Join(root, "unavailable")}
				cfg.Thresholds = Thresholds{WindowHours: 24, TailBytes: 4096, ExcerptChars: 1000}
				cfg.Vaults = map[string]string{checkout: vault}
				if kind == "facts without judgments" {
					profile := filepath.Join(root, "profile")
					writeFile(t, filepath.Join(profile, "projects", "fixture", sessAlpha+".jsonl"), fmt.Sprintf(`{"type":"assistant","cwd":%q,"message":{"role":"assistant","content":"synthetic turn"}}`, checkout)+"\n", testNow)
					cfg.Profiles = []Profile{{Name: "fixture", Kind: "claude", Dir: filepath.Join(profile, "projects")}}
					snapshot := collect(t.Context(), cfg, nil, testNow)
					require.Len(t, snapshot.Projects, 1)
					assert.Equal(t, checkout, snapshot.Projects[0].Root)
					assert.Empty(t, snapshot.Judgments)
					assert.False(t, cfg.Gate.TodoContextEnabled)
					return
				}
				todo.Links[0].slug = "example/repo"
				todo.Links = append(todo.Links, Link{Kind: "commit", SHA: sha, State: "unchecked"})
				tabs := []Tab{{ID: "fixture-tab", Label: "fixture", Agent: "codex", State: "working", CWD: checkout}}
				collector := &collector{cfg: cfg, r: runner{cmds: cfg.Commands}}
				grouped := collector.groupProjects(t.Context(), tabs, nil, []Todo{todo})
				var projects []Project
				for _, project := range grouped {
					projects = append(projects, *project)
				}
				states := make([]todoState, 2)
				errs := make([]error, 2)
				parallel([]Todo{todo, todo}, func(i int, item Todo) {
					states[i], errs[i] = evaluationState(t.Context(), cfg, projects, item, "2026-10-12")
				})
				for _, err := range errs {
					require.NoError(t, err)
				}
				assert.Equal(t, "unchecked", todo.Links[1].State)
				state := states[0]
				assert.Equal(t, states[0], states[1])
				require.Len(t, state.WikiPages, 1)
				assert.Equal(t, text, state.WikiPages[0].Text)
				assert.Equal(t, "on_default", state.Links[1].State)
				assert.Equal(t, []tabBrief{{Label: "fixture", Agent: "codex", State: "working"}}, state.ProjectTabs)
				assert.False(t, cfg.Gate.TodoContextEnabled)
				return
			}
			if tt.reader != "" {
				input := filepath.Join(root, "private input")
				path := filepath.Join(input, tt.path)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
				require.NoError(t, os.WriteFile(path, []byte("synthetic private fixture; "+todo.UUID), 0o600))
				requested := path
				if tt.symlink {
					requested = filepath.Join(input, "alias.md")
					require.NoError(t, os.Symlink(path, requested))
				}
				if tt.reader == "wiki" {
					cfg.Vaults["work"] = input
					page, err := wikiPage(cfg, input, requested)
					require.ErrorContains(t, err, "sensitive path")
					if tt.symlink {
						assert.Contains(t, err.Error(), "resolves to a sensitive path")
					}
					assert.Empty(t, page.Text)
				} else {
					cfg.Context.HandoffDir = input
					todo.Annotations = nil
					state, err := todoContext(cfg, &Project{}, todo, nil, "2026-10-12")
					if filepath.Ext(path) != ".md" && !tt.symlink {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, "sensitive path")
						if tt.symlink {
							assert.Contains(t, err.Error(), "resolves to a sensitive path")
						}
					}
					assert.Empty(t, state.Handoffs)
				}
				return
			}
			var wantError string
			switch kind {
			case "non-ASCII page", "spaced filename":
				name := "日本語-note.md"
				if kind == "spaced filename" {
					name = "team plans.md"
				}
				require.NoError(t, os.WriteFile(filepath.Join(vault, "projects", name), []byte(text), 0o600))
				todo.Annotations[0] = "wiki: projects/" + name
			case "missing page":
				todo.Annotations = []string{"wiki: absent"}
				wantError = "found 0"
			case "ambiguous name":
				require.NoError(t, os.WriteFile(filepath.Join(vault, "project.md"), []byte("another"), 0o600))
				wantError = "found 2"
			case "cross-vault symlink":
				require.NoError(t, os.WriteFile(filepath.Join(foreign, "foreign.md"), []byte("foreign"), 0o600))
				require.NoError(t, os.Remove(page))
				require.NoError(t, os.Symlink(filepath.Join(foreign, "foreign.md"), page))
				wantError = "outside"
			case "missing handoff directory":
				cfg.Context.HandoffDir = filepath.Join(root, "absent")
				wantError = "handoff directory"
			}
			state, err := todoContext(cfg, &Project{}, todo, nil, "2026-10-12")
			if wantError != "" {
				require.ErrorContains(t, err, wantError)
				return
			}
			require.NoError(t, err)
			require.Len(t, state.WikiPages, 1)
			assert.Equal(t, text, state.WikiPages[0].Text)
			assert.Len(t, state.Handoffs, 3)
			assert.Equal(t, cfg.OwnerPolicy, state.OwnerPolicy)
			assert.Equal(t, todo.Annotations[:1], state.Annotations)
			data, err := json.Marshal(state)
			require.NoError(t, err)
			assert.NotContains(t, string(data), "transcript_excerpt")
		})
	}
}

func TestOwnerConfidencePolicy(t *testing.T) {
	cases := []struct {
		label      string
		confidence float64
		verdict    string
	}{
		{"cto", .6, verdictAct}, {"cos", .6, verdictAct}, {"cto", .59, verdictEscalate}, {"user", 1, verdictEscalate}, {"unknown", 1, verdictEscalate},
	}
	for _, tt := range cases {
		t.Run(tt.label+" "+tt.verdict, func(t *testing.T) {
			gate := Gate{th: Thresholds{Floor: .6, Low: .75, High: .85}}
			verdict, _ := gate.verdict(gateTodoOwner, tt.label, tt.confidence)
			assert.Equal(t, tt.verdict, verdict)
		})
	}
}

func TestDisabledTodoGates(t *testing.T) {
	cases := []struct {
		name, owner, status string
		reason              string
	}{
		{"state disabled", "cto", "pending", "disabled pending evaluation"}, {"owner disabled", "", "pending", "no assignee tag"}, {"waiting not assigned", "", "waiting", "no assignee tag"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			project := &Project{Todos: []Todo{{UUID: "todo", Status: tt.status, Assignee: tt.owner}}}
			judgments := judgeAll(t.Context(), &Gate{}, []*Project{project}, testNow, Config{})
			require.Len(t, judgments, 1)
			assert.Equal(t, verdictEscalate, judgments[0].Verdict)
			assert.Contains(t, judgments[0].Reason, tt.reason)
		})
	}
}

func TestInputLimitErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		detail string
		limit  bool
	}{
		{"token budget", 422, "state plus question exceeds 32768 tokens", true}, {"body budget", 413, "", true}, {"other validation", 422, "missing required question", false}, {"server failure", 500, "token error", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.limit, inputLimitError(&gateHTTPError{Status: tt.status, Detail: tt.detail}))
		})
	}
}
