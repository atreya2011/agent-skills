package main

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
		{"backtick reference", "wiki: `articles/日本語-note.md`", []string{"articles/日本語-note.md"}},
		{"double quoted reference", "wiki: \"articles/team plans.md\"", []string{"articles/team plans.md"}},
		{"single quoted reference", "wiki: 'team plans/article.md'", []string{"team plans/article.md"}},
		{"wiki link reference", "wiki: [[articles/team plans.md]]", []string{"articles/team plans.md"}},
		{"separate references", "wiki: a.md b.md", []string{"a.md", "b.md"}},
		{"mixed spaced Unicode references", "wiki: `articles/team plans.md` \"articles/日本語-note.md\" [[team plans/日本語 note.md]]", []string{"articles/team plans.md", "articles/日本語-note.md", "team plans/日本語 note.md"}},
		{"unformatted spaced Unicode references", "wiki: articles/team plans.md articles/日本語-note.md", []string{"articles/team plans.md", "articles/日本語-note.md"}},
		{"apostrophe filename", "wiki: team's notes.md", []string{"team's notes.md"}},
		{"brackets filename", "wiki: notes [draft].md", []string{"notes [draft].md"}},
		{"leading brackets filename", "wiki: [draft] notes.md", []string{"[draft] notes.md"}},
		{"literal opening quotes", "wiki: \"draft\" notes.md", []string{"\"draft\" notes.md"}},
		{"literal opening backticks", "wiki: `draft` notes.md", []string{"`draft` notes.md"}},
		{"literal opening wiki brackets", "wiki: [[draft]] notes.md", []string{"[[draft]] notes.md"}},
		{"quotes filename", "wiki: notes \"draft\".md", []string{"notes \"draft\".md"}},
		{"backticks filename", "wiki: notes `draft`.md", []string{"notes `draft`.md"}},
		{"parentheses filename", "wiki: notes (draft).md", []string{"notes (draft).md"}},
		{"comma filename", "wiki: notes, draft.md", []string{"notes, draft.md"}},
		{"semicolon filename", "wiki: notes; draft.md", []string{"notes; draft.md"}},
		{"pipe filename", "wiki: notes|draft.md", []string{"notes|draft.md"}},
		{"quoted apostrophe filename", "wiki: 'team's notes.md'", []string{"team's notes.md"}},
		{"quoted quote filename", "wiki: \"team \"draft\" notes.md\"", []string{"team \"draft\" notes.md"}},
		{"wrapped bracket filename", "wiki: [[notes [draft].md]]", []string{"notes [draft].md"}},
		{"wrapped mixed punctuation", "wiki: [[日本語/team's \"draft\" `v2` [plan] (review), final; note.md]]", []string{"日本語/team's \"draft\" `v2` [plan] (review), final; note.md"}},
		{"nested markdown directory", "wiki: archive.md/article.md", []string{"archive.md/article.md"}},
		{"nested markdown suffix", "wiki: archive.md-backup/article.md", []string{"archive.md-backup/article.md"}},
		{"nested markdown directories", "wiki: archive.md/history.md/article.md:95", []string{"archive.md/history.md/article.md:95"}},
		{"wrapped nested markdown directory", "wiki: [[archive.md/team's [draft] notes.md]]", []string{"archive.md/team's [draft] notes.md"}},
		{"wrapped spaced extensions", "wiki: \"a.md b.md\"", []string{"a.md b.md"}},
		{"wrapped locator", "wiki: `team's notes.md:95`", []string{"team's notes.md:95"}},
		{"multiple punctuated references", "wiki: a.md, team's notes.md; [[archive.md/notes [draft].md]]", []string{"a.md", "archive.md/notes [draft].md", "team's notes.md"}},
		{"locator-shaped directory", "wiki: archive.md:95/article.md", []string{"archive.md:95/article.md"}},
		{"single bracket reference", "wiki: [notes.md]", []string{"notes.md"}},
		{"single bracket punctuated reference", "wiki: [notes [draft].md]", []string{"notes [draft].md"}},
		{"compact metadata", "wiki: build-upgrade-schedule, collaborator| src: work/page.md", []string{"build-upgrade-schedule", "collaborator"}},
		{"compact metadata marker", "wiki: build-upgrade-schedule, collaborator|src: work/page.md", []string{"build-upgrade-schedule", "collaborator"}},
		{"quoted metadata text filename", "wiki: \"notes |src: draft.md\"", []string{"notes |src: draft.md"}},
		{"double backtick reference", "wiki: ``notes.md``", []string{"notes.md"}},
		{"double backtick internal tick", "wiki: ``notes `draft`.md``", []string{"notes `draft`.md"}},
		{"triple backtick internal ticks", "wiki: ```notes ``draft``.md```", []string{"notes ``draft``.md"}},
		{"wiki wrapper repository metadata", "wiki: [[notes.md]]. Repo: project", []string{"notes.md"}},
		{"wiki wrapper contract metadata", "wiki: [[notes.md]]. Contract: approved", []string{"notes.md"}},
		{"wiki wrapper other metadata", "wiki: [[notes.md]]. Owner: team", []string{"notes.md"}},
		{"bracket wrapper metadata", "wiki: [notes.md]. Repo: project. Contract: approved", []string{"notes.md"}},
		{"backtick wrapper metadata", "wiki: `notes.md`. Repo: project", []string{"notes.md"}},
		{"double backtick wrapper metadata", "wiki: ``notes.md``. Contract: approved", []string{"notes.md"}},
		{"quoted wrapper metadata", "wiki: \"notes.md\". Repo: project", []string{"notes.md"}},
		{"single quoted wrapper metadata", "wiki: 'notes.md'. Owner: team", []string{"notes.md"}},
		{"plain reference metadata", "wiki: notes.md. Repo: project. Contract: approved", []string{"notes.md"}},
		{"plain label destination", "wiki: notes (articles/note.md)", []string{"articles/note.md"}},
		{"markdown label destination", "wiki: [notes](articles/note.md)", []string{"articles/note.md"}},
		{"label destination locator", "wiki: notes (articles/note.md:95)", []string{"articles/note.md:95"}},
		{"multiple destinations", "wiki: notes (articles/note.md); [other](articles/other.md)", []string{"articles/note.md", "articles/other.md"}},
		{"nested metadata", "wiki: a.md (Origin (section), details); b.md", []string{"a.md", "b.md"}},
		{"deep metadata", "wiki: a.md (Origin (section (detail)), details); b.md", []string{"a.md", "b.md"}},
		{"nested metadata comma separator", "wiki: a.md (Origin (section), details), b.md", []string{"a.md", "b.md"}},
		{"nested metadata terminal", "wiki: a.md (Origin (section), details)", []string{"a.md"}},
		{"formatted nested metadata", "wiki: [[a.md]] (Origin (section), details); `b.md`", []string{"a.md", "b.md"}},
		{"plain before link", "wiki: a.md notes (articles/note.md)", []string{"a.md", "articles/note.md"}},
		{"wrapped before link", "wiki: [[a.md]] notes (articles/note.md)", []string{"a.md", "articles/note.md"}},
		{"link before plain", "wiki: notes (articles/note.md) a.md", []string{"a.md", "articles/note.md"}},
		{"nested path before link", "wiki: archive.md/article.md notes (articles/note.md)", []string{"archive.md/article.md", "articles/note.md"}},
		{"nested notes before link", "wiki: a.md (Origin (section), details); notes (articles/note.md)", []string{"a.md", "articles/note.md"}},
		{"own question excluded", "ask: wiki: old-verdict", nil},
	}
	forms := []struct{ name, format, path string }{
		{"plain", "%s", "%s"}, {"wiki", "[[%s]]", "%s"}, {"bracket", "[%s]", "%s"},
		{"backtick", "`%s`", "%s"}, {"double backtick", "``%s``", "%s"},
		{"double quote", "\"%s\"", "%s"}, {"single quote", "'%s'", "%s"},
		{"label", "notes (%s)", "%s"}, {"markdown label", "[notes](%s)", "%s"},
		{"nested", "%s", "archive.md/%s"}, {"locator", "%s", "%s:95"},
	}
	for _, first := range forms {
		for _, second := range forms {
			for _, separator := range []string{" ", ", ", "; "} {
				left, right := fmt.Sprintf(first.path, "a.md"), fmt.Sprintf(second.path, "b.md")
				want := []string{left, right}
				slices.Sort(want)
				cases = append(cases, struct {
					name, annotation string
					want             []string
				}{"compatibility " + first.name + " then " + second.name + " separator " + separator,
					"wiki: " + fmt.Sprintf(first.format, left) + separator + fmt.Sprintf(second.format, right), want})
			}
		}
	}
	for _, first := range forms {
		for depth := 1; depth <= 3; depth++ {
			for _, separator := range []string{" ", ", ", "; "} {
				left := fmt.Sprintf(first.path, "a.md")
				want := []string{left, "b.md"}
				slices.Sort(want)
				note := strings.Repeat("(Origin ", depth) + "section" + strings.Repeat(")", depth)
				cases = append(cases, struct {
					name, annotation string
					want             []string
				}{fmt.Sprintf("balanced %s depth %d separator %s", first.name, depth, separator),
					"wiki: " + fmt.Sprintf(first.format, left) + " " + note + separator + "b.md", want})
			}
		}
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
	for _, name := range []string{"complete context", "checkout evaluation", "facts without judgments", "non-ASCII page", "spaced filename", "apostrophe filename", "brackets filename", "nested markdown directory", "locator-shaped directory", "missing page", "ambiguous name", "cross-vault symlink", "missing handoff directory"} {
		cases = append(cases, struct {
			name, path, reader string
			symlink            bool
		}{name: name})
	}
	for _, file := range []struct{ name, path string }{
		{"environment", ".env"}, {"local environment", ".env.local"}, {"production environment", ".env.production"},
		{"environment directory", ".env.local/page.md"}, {"environment markdown", ".env.local.md"}, {"private key", "signing.key"}, {"certificate", "signing.pem"},
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
				require.NoError(t, os.MkdirAll(input, 0o700))
				path := filepath.Join(input, tt.path)
				if tt.reader == "handoff" && tt.symlink {
					path = filepath.Join(root, "private target", tt.path)
				}
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
			case "non-ASCII page", "spaced filename", "apostrophe filename", "brackets filename", "nested markdown directory", "locator-shaped directory":
				name := map[string]string{
					"non-ASCII page": "日本語-note.md", "spaced filename": "team plans.md",
					"apostrophe filename": "team's notes.md", "brackets filename": "notes [draft].md",
					"nested markdown directory": "archive.md/article.md", "locator-shaped directory": "archive.md:95/article.md",
				}[kind]
				require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(vault, "projects", name)), 0o700))
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
