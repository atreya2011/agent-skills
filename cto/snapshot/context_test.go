package main

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		{"own question excluded", "ask: wiki: old-verdict", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.want, wikiNames([]string{tt.annotation})) })
	}
}

func TestTodoContextFiles(t *testing.T) {
	for _, kind := range []string{"complete context", "missing page", "ambiguous name", "cross-vault symlink", "missing handoff directory"} {
		t.Run(kind, func(t *testing.T) {
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
			var wantError string
			switch kind {
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
			assert.Equal(t, []string{"wiki: project"}, state.Annotations)
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

func TestContextSensitiveFiles(t *testing.T) {
	cases := []struct {
		name, path string
	}{
		{"environment", ".env"}, {"private key", "signing.key"}, {"certificate", "signing.pem"},
		{"private metadata", "credentials.md"}, {"private directory", "credentials/page.md"},
	}
	for _, tt := range cases {
		for _, reader := range []string{"wiki", "handoff"} {
			for _, link := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/symlink=%t", tt.name, reader, link), func(t *testing.T) {
					root := t.TempDir()
					path := filepath.Join(root, tt.path)
					require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
					require.NoError(t, os.WriteFile(path, []byte("synthetic private fixture; todo marker"), 0o600))
					requested := path
					if link {
						requested = filepath.Join(root, "alias.md")
						require.NoError(t, os.Symlink(path, requested))
					}
					cfg := Config{Vaults: map[string]string{"test": root}}
					if reader == "wiki" {
						page, err := wikiPage(cfg, root, requested)
						require.ErrorContains(t, err, "sensitive path")
						if link {
							assert.Contains(t, err.Error(), "resolves to a sensitive path")
						}
						assert.Empty(t, page.Text)
					} else {
						cfg.Context.HandoffDir = root
						state, err := todoContext(cfg, &Project{}, Todo{UUID: "todo marker"}, nil, "2026-10-12")
						if filepath.Ext(path) != ".md" && !link {
							require.NoError(t, err)
						} else {
							require.ErrorContains(t, err, "sensitive path")
						}
						assert.Empty(t, state.Handoffs)
					}
				})
			}
		}
	}
}
