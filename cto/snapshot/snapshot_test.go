package main

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// Todo UUIDs of the recorded task export.
const (
	todoParser    = "00000000-0000-4000-8000-000000000001" // linked pull request is merged
	todoDocs      = "00000000-0000-4000-8000-000000000002" // linked issue is open and ready
	todoDentist   = "00000000-0000-4000-8000-000000000003" // +cos
	todoFlake     = "00000000-0000-4000-8000-000000000004" // untagged, linked issue is closed
	todoRetry     = "00000000-0000-4000-8000-000000000005" // cites a session
	todoMerged    = "00000000-0000-4000-8000-000000000007" // cites a commit on the default branch
	todoGamma     = "00000000-0000-4000-8000-000000000008" // its issue cannot be read
	todoUnmerged  = "00000000-0000-4000-8000-000000000010" // cites a commit on a feature branch
	wantGateCalls = 7
)

type scriptedAnswer struct {
	label      string
	confidence float64
}

// baseScript is the gate's answer for every subject that reaches a gate in the
// base world; rules decide the other todos and the CoS todo is not judged.
var baseScript = map[string]scriptedAnswer{
	"todo_state:" + todoDocs:          {"open", 0.9},
	"todo_state:" + todoRetry:         {"done", 0.85},
	"todo_state:" + todoUnmerged:      {"stale", 0.7},
	"session_activity:" + sessAlpha:   {"implementing", 0.9},
	"session_activity:" + sessNotes:   {"blocked", 0.75},
	"session_activity:" + sessBeta:    {"investigating", 0.65},
	"session_activity:" + sessUnbound: {"unknown", 0.9},
	"orchestrator_reply:":             {"done", 0.85},
}

var gateDefs = map[string]gateDef{
	gateTodoState.Name: gateTodoState, gateSession.Name: gateSession, gateReply.Name: gateReply,
}

// scriptedGate replays the script by gate name and the state's id, and answers
// 500 to anything the script does not cover.
func scriptedGate(t *testing.T, script map[string]scriptedAnswer) *fakeGate {
	t.Helper()
	return newFakeGate(t, nil, func(f *fakeGate, body []byte, w http.ResponseWriter) {
		var req struct {
			State struct {
				ID string `json:"id"`
			} `json:"state"`
			Questions map[string]any `json:"questions"`
		}
		if err := json.Unmarshal(body, &req); err != nil || len(req.Questions) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for name := range req.Questions {
			a, ok := script[name+":"+req.State.ID]
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = io.WriteString(w, answerBody(name, a.label, probs(gateDefs[name], a.label, 0.9), a.confidence))
		}
	})
}

// seen lists the "gate:id" of every request the fake gate received.
func (f *fakeGate) seen(t *testing.T) []string {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, b := range f.bodies {
		var req struct {
			State struct {
				ID string `json:"id"`
			} `json:"state"`
			Questions map[string]any `json:"questions"`
		}
		require.NoError(t, json.Unmarshal(b, &req))
		for name := range req.Questions {
			out = append(out, name+":"+req.State.ID)
		}
	}
	slices.Sort(out)
	return out
}

// runTool runs the program against the world and returns its exit code and
// output. The OS keyring is replaced with an empty in-memory one.
func runTool(t *testing.T, w *world, stdin string, args ...string) (int, string, string) {
	t.Helper()
	keyring.MockInit()
	var out, errOut bytes.Buffer
	code := run(t.Context(), append(args, "--local-file", w.Local), strings.NewReader(stdin), &out, &errOut, testNow)
	return code, out.String(), errOut.String()
}

var (
	hashField  = regexp.MustCompile(`"hash":\s*"[0-9a-f]{16}"`)
	hashInNote = regexp.MustCompile(` [0-9a-f]{8} (\d{4}-\d{2}-\d{2})"`)
	commitID   = regexp.MustCompile(`\b[0-9a-f]{40}\b`)
)

// normalize replaces what differs between runs: the temporary root, the commit
// IDs and the request hashes, which cover paths under the root.
func normalize(s string, w *world) string {
	s = strings.NewReplacer(w.Root, "{{root}}", w.ShaMerged, "{{sha_merged}}", w.ShaOpen, "{{sha_unmerged}}").Replace(s)
	s = commitID.ReplaceAllString(s, "{{sha}}")
	s = hashField.ReplaceAllString(s, `"hash":"HASH"`)
	return hashInNote.ReplaceAllString(s, ` HASH8 $1"`)
}

func decode(t *testing.T, out string) Snapshot {
	t.Helper()
	var s Snapshot
	require.NoError(t, json.Unmarshal([]byte(out), &s), out)
	return s
}

func (s Snapshot) judgment(subject, id string) Judgment {
	for _, j := range s.Judgments {
		if j.Subject == subject && j.ID == id {
			return j
		}
	}
	return Judgment{}
}

func TestSnapshotOfTheBaseWorld(t *testing.T) {
	f := scriptedGate(t, baseScript)
	w := newWorld(t, f.srv.URL)
	t.Setenv(keyEnv, keySentinel)

	code, out, errOut := runTool(t, w, "")
	require.Equal(t, 0, code, errOut)
	want, err := os.ReadFile(filepath.Join("testdata", "base", "expected.json"))
	require.NoError(t, err)
	assert.JSONEq(t, normalize(string(want), w), normalize(out, w))

	t.Run("rules decide without a gate call", func(t *testing.T) {
		s := decode(t, out)
		for id, rule := range map[string]string{todoParser: "pr_merged", todoFlake: "issue_closed", todoMerged: "commit_on_default"} {
			j := s.judgment("todo", id)
			assert.Equal(t, decidedByRule, j.DecidedBy, id)
			assert.Equal(t, rule, j.Rule, id)
			assert.Equal(t, verdictAct, j.Verdict, id)
			assert.Equal(t, "done", j.Label, id)
		}
		assert.NotContains(t, strings.Join(f.seen(t), " "), todoParser)
		assert.NotContains(t, strings.Join(f.seen(t), " "), todoFlake)
		assert.NotContains(t, strings.Join(f.seen(t), " "), todoMerged)
		assert.Len(t, f.seen(t), wantGateCalls, "the CoS todo, the completed todo and the rule-decided todos make no call")
	})

	t.Run("a todo that cites a session outside the window sends that session's excerpt", func(t *testing.T) {
		f.mu.Lock()
		defer f.mu.Unlock()
		var retry []byte
		for _, b := range f.bodies {
			if bytes.Contains(b, []byte(todoRetry)) {
				retry = b
			}
		}
		require.NotNil(t, retry)
		assert.Contains(t, string(retry), "Limits tuned to 5 attempts.")
	})
}

func TestSnapshotReportsUnreadableSourcesAndCarriesOn(t *testing.T) {
	tests := []struct {
		name       string
		break_     func(t *testing.T, w *world)
		wantSource string
		wantTarget string
		check      func(t *testing.T, w *world, s Snapshot)
	}{
		{"herdr workspace list", func(t *testing.T, w *world) { removeRecording(t, w, "herdr", "workspace_list") }, "herdr workspace list", "",
			func(t *testing.T, _ *world, s Snapshot) {
				assert.Empty(t, s.Workspaces)
				assert.NotEmpty(t, s.Projects)
			}},
		{"herdr agent list", func(t *testing.T, w *world) { removeRecording(t, w, "herdr", "agent_list") }, "herdr agent list", "",
			func(t *testing.T, _ *world, s Snapshot) {
				for _, p := range s.Projects {
					for _, tab := range p.Tabs {
						assert.Empty(t, tab.Agent)
					}
				}
			}},
		{"taskwarrior export", func(t *testing.T, w *world) { removeRecording(t, w, "task", "export") }, "task export", "",
			func(t *testing.T, _ *world, s Snapshot) {
				for _, j := range s.Judgments {
					assert.Equal(t, "session", j.Subject, "no todo can be judged without the export")
				}
				assert.NotEmpty(t, s.Judgments)
			}},
		{"the git binary", func(t *testing.T, w *world) { setCommand(t, w, "git", filepath.Join(w.Root, "no-git")) }, "git", filepath.Join("", "alpha-wt"),
			func(t *testing.T, w *world, s Snapshot) {
				ids := []string{}
				for _, p := range s.Projects {
					ids = append(ids, p.ID)
				}
				assert.Contains(t, ids, filepath.Join(w.Root, "alpha-wt"), "without git a directory is its own project")
			}},
		{"gh pr list", func(t *testing.T, w *world) {
			failRecording(t, w, "gh", "pr_list_--repo_example-alpha_--state_all_--limit_30")
		}, "gh pr list", "example/alpha",
			func(t *testing.T, _ *world, s Snapshot) {
				for _, p := range s.Projects {
					if p.ID == "example/alpha" {
						assert.Empty(t, p.PRs)
						assert.NotEmpty(t, p.Todos)
					}
				}
			}},
		{"a linked pull request", func(t *testing.T, w *world) {
			removeRecording(t, w, "gh", "pr_view_https---github.com-example-alpha-pull-7")
		}, "gh pr view", "https://github.com/example/alpha/pull/7",
			func(t *testing.T, _ *world, s Snapshot) {
				j := s.judgment("todo", todoParser)
				assert.Equal(t, verdictEscalate, j.Verdict)
				assert.Equal(t, decidedByError, j.DecidedBy)
				assert.Contains(t, j.Reason, "unreadable")
			}},
		{"a profile directory", func(t *testing.T, w *world) {
			require.NoError(t, os.RemoveAll(filepath.Join(w.Root, "codex-main")))
		}, "transcripts", "codex",
			func(t *testing.T, _ *world, s Snapshot) {
				assert.Contains(t, s.Profiles[2].Name, "codex")
				assert.Zero(t, s.Profiles[2].SessionsInWindow)
			}},
		{"the vault", func(t *testing.T, w *world) { require.NoError(t, os.RemoveAll(filepath.Join(w.Root, "vaults"))) }, "vault", "",
			func(t *testing.T, _ *world, s Snapshot) {
				for _, p := range s.Projects {
					if p.ID == "example/alpha" {
						require.NotNil(t, p.Vault)
						assert.False(t, p.Vault.Readable)
					}
				}
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := scriptedGate(t, baseScript)
			w := newWorld(t, f.srv.URL)
			tt.break_(t, w)
			code, out, errOut := runTool(t, w, "")
			require.Equal(t, 0, code, "a missing source is reported, not fatal: %s", errOut)
			s := decode(t, out)
			found := slices.ContainsFunc(s.Unreadable, func(u Unreadable) bool {
				return u.Source == tt.wantSource && strings.Contains(u.Target, tt.wantTarget)
			})
			assert.True(t, found, "unreadable should list %q %q, got %+v", tt.wantSource, tt.wantTarget, s.Unreadable)
			tt.check(t, w, s)
		})
	}
}

func removeRecording(t *testing.T, w *world, cmd, key string) {
	t.Helper()
	require.NoError(t, os.Remove(filepath.Join(w.Fixtures, cmd, key+".out")))
}

func failRecording(t *testing.T, w *world, cmd, key string) {
	t.Helper()
	writeFile(t, filepath.Join(w.Fixtures, cmd, key+".code"), "1\n", time.Time{})
	writeFile(t, filepath.Join(w.Fixtures, cmd, key+".err"), "HTTP 502\n", time.Time{})
}

// setCommand adds a program path to the local file's commands table.
func setCommand(t *testing.T, w *world, name, path string) {
	t.Helper()
	data, err := os.ReadFile(w.Local)
	require.NoError(t, err)
	edited := strings.Replace(string(data), "[commands]\n", fmt.Sprintf("[commands]\n%s = %q\n", name, path), 1)
	require.NoError(t, os.WriteFile(w.Local, []byte(edited), 0o600))
}

func TestSnapshotEscalatesWhenTheGateFails(t *testing.T) {
	tests := []struct {
		name  string
		reply func(int, http.ResponseWriter)
	}{
		{"service error", serve(http.StatusInternalServerError, "")},
		{"malformed response", serve(http.StatusOK, `{"answers": {`)},
		{"wrong labels", serve(http.StatusOK, answerBody("todo_state", "done", map[string]float64{"done": 1}, 1))},
		{"unauthorized", serve(http.StatusUnauthorized, `{"error":"bad key"}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeGate(t, tt.reply, nil)
			w := newWorld(t, f.srv.URL)
			t.Setenv(keyEnv, keySentinel)
			code, out, _ := runTool(t, w, "")
			require.Equal(t, 0, code)
			s := decode(t, out)
			for _, j := range s.Judgments {
				if j.DecidedBy == decidedByRule {
					assert.Equal(t, verdictAct, j.Verdict, "a rule needs no gate")
					continue
				}
				assert.Equal(t, verdictEscalate, j.Verdict, "%s %s", j.Subject, j.ID)
				assert.Equal(t, decidedByError, j.DecidedBy)
				assert.NotEmpty(t, j.Reason)
			}
		})
	}
}

func TestSnapshotKeyHandling(t *testing.T) {
	t.Run("the key is out of the environment of every source command and stays out of every output", func(t *testing.T) {
		f := scriptedGate(t, baseScript)
		w := newWorld(t, f.srv.URL)
		envLog := filepath.Join(w.Root, "env.log")
		t.Setenv("SHIM_ENVLOG", envLog)
		t.Setenv(keyEnv, keySentinel)

		code, out, errOut := runTool(t, w, "")
		require.Equal(t, 0, code, errOut)
		children, err := os.ReadFile(envLog)
		require.NoError(t, err)
		assert.NotContains(t, string(children), keyEnv)
		assert.NotContains(t, string(children), keySentinel)
		logged, err := os.ReadFile(filepath.Join(w.Root, "state", "gate.log"))
		require.NoError(t, err)
		for name, text := range map[string]string{"stdout": out, "stderr": errOut, "gate log": string(logged)} {
			assert.NotContains(t, text, keySentinel, name)
		}
		require.NotEmpty(t, f.headers)
		assert.Equal(t, "Bearer "+keySentinel, f.headers[0].Get("Authorization"), "the gate still received the key")
	})

	t.Run("the keyring supplies the key when the environment has none", func(t *testing.T) {
		f := scriptedGate(t, baseScript)
		w := newWorld(t, f.srv.URL)
		t.Setenv(keyEnv, "")
		keyring.MockInit()
		require.NoError(t, keyring.Set(keyringService, keyringUser, keySentinel))
		var out, errOut bytes.Buffer
		code := run(t.Context(), []string{"--local-file", w.Local}, strings.NewReader(""), &out, &errOut, testNow)
		require.Equal(t, 0, code, errOut.String())
		require.NotEmpty(t, f.headers)
		assert.Equal(t, "Bearer "+keySentinel, f.headers[0].Get("Authorization"))
	})

	t.Run("a keyring miss with no environment key escalates every gate and sends nothing", func(t *testing.T) {
		f := scriptedGate(t, baseScript)
		w := newWorld(t, f.srv.URL)
		t.Setenv(keyEnv, "")
		code, out, _ := runTool(t, w, "")
		require.Equal(t, 0, code)
		s := decode(t, out)
		noKey := 0
		for _, j := range s.Judgments {
			if j.DecidedBy != decidedByRule {
				assert.Equal(t, verdictEscalate, j.Verdict, "%s %s", j.Subject, j.ID)
			}
			if strings.Contains(j.Reason, "no API key") {
				noKey++
			}
		}
		assert.Equal(t, wantGateCalls, noKey, "every judgment that would have called a gate says why it did not")
		assert.Zero(t, f.requests())
	})
}

func TestReplySubcommand(t *testing.T) {
	tests := []struct {
		name        string
		stdin       string
		script      map[string]scriptedAnswer
		wantVerdict string
		wantLabel   string
		wantCalls   int
	}{
		{"a confident done acts", "All merged. The tabs can be closed.", map[string]scriptedAnswer{"orchestrator_reply:": {"done", 0.85}}, verdictAct, "done", 1},
		{"a done below the high threshold escalates", "I think we are done.", map[string]scriptedAnswer{"orchestrator_reply:": {"done", 0.7}}, verdictEscalate, "done", 1},
		{"an unclear reply escalates", "Hmm.", map[string]scriptedAnswer{"orchestrator_reply:": {"unclear", 0.99}}, verdictEscalate, "unclear", 1},
		{"a not_done reply acts at the low threshold", "Still reviewing.", map[string]scriptedAnswer{"orchestrator_reply:": {"not_done", 0.65}}, verdictAct, "not_done", 1},
		{"an empty reply escalates without a call", "", nil, verdictEscalate, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := scriptedGate(t, tt.script)
			w := newWorld(t, f.srv.URL)
			t.Setenv(keyEnv, keySentinel)
			code, out, errOut := runTool(t, w, tt.stdin, "reply")
			require.Equal(t, 0, code, errOut)
			var j Judgment
			require.NoError(t, json.Unmarshal([]byte(out), &j))
			assert.Equal(t, "reply", j.Subject)
			assert.Equal(t, tt.wantVerdict, j.Verdict)
			assert.Equal(t, tt.wantLabel, j.Label)
			assert.Equal(t, tt.wantCalls, f.requests())
		})
	}
}

func TestRunRejectsABadInvocationOrLocalFile(t *testing.T) {
	w := newWorld(t, "")
	data, err := os.ReadFile(w.Local)
	require.NoError(t, err)
	tests := []struct {
		name string
		edit func(path string)
		args []string
	}{
		{"an unknown flag", func(string) {}, []string{"--bogus"}},
		{"a local file that is missing", func(p string) { require.NoError(t, os.Remove(p)) }, nil},
		{"a local file with an unknown field", func(p string) {
			require.NoError(t, os.WriteFile(p, append(data, []byte("```toml\n[gate]\napi_key = \"x\"\n```\n")...), 0o600))
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t, "")
			tt.edit(w.Local)
			var out, errOut bytes.Buffer
			code := run(t.Context(), append(slices.Clone(tt.args), "--local-file", w.Local), strings.NewReader(""), &out, &errOut, testNow)
			assert.Equal(t, 2, code)
			assert.Empty(t, out.String())
			assert.NotEmpty(t, errOut.String())
		})
	}
}
