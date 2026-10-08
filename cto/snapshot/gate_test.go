package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

const keySentinel = "tsk-test-sentinel-0123456789"

var testNow = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

var testThresholds = Thresholds{Floor: 0.6, Low: 0.6, High: 0.8, WindowHours: 6, TailBytes: 4096, ExcerptChars: 400}

// fakeGate replays scripted responses and records the requests it receives.
type fakeGate struct {
	srv     *httptest.Server
	mu      sync.Mutex
	bodies  [][]byte
	headers []http.Header
	reply   func(n int, w http.ResponseWriter)
	handler func(f *fakeGate, body []byte, w http.ResponseWriter)
}

// newFakeGate starts a fake gate. reply answers by request number; a handler,
// when given, answers by request content instead.
func newFakeGate(t *testing.T, reply func(n int, w http.ResponseWriter), handler ...func(f *fakeGate, body []byte, w http.ResponseWriter)) *fakeGate {
	t.Helper()
	f := &fakeGate{reply: reply}
	if len(handler) > 0 {
		f.handler = handler[0]
	}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.bodies = append(f.bodies, body)
		f.headers = append(f.headers, r.Header.Clone())
		n := len(f.bodies)
		f.mu.Unlock()
		if f.handler != nil {
			f.handler(f, body, w)
			return
		}
		f.reply(n, w)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeGate) requests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bodies)
}

// answerBody is a documented-shape response with one choice answer.
func answerBody(gate, choice string, probs map[string]float64, confidence float64) string {
	p, _ := json.Marshal(probs, json.Deterministic(true))
	return fmt.Sprintf(`{"model":"jev-1.13.0","answers":{%q:{"type":"choice","choice":%q,"probabilities":%s,"confidence":%v}},"usage":{"input_tokens":1,"output_tokens":1}}`,
		gate, choice, p, confidence)
}

func serve(status int, body string) func(int, http.ResponseWriter) {
	return func(_ int, w http.ResponseWriter) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

func testGate(t *testing.T, url string) (*Gate, string) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "state", "gate.log")
	logger, f, err := openGateLog(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	g := newGate(GateConfig{URL: url}, testThresholds, keySentinel, nil, logger, testNow)
	g.client.RetryWaitMin = time.Millisecond
	g.client.RetryWaitMax = 5 * time.Millisecond
	return g, logPath
}

func probs(gate gateDef, top string, p float64) map[string]float64 {
	out := map[string]float64{}
	rest := (1 - p) / float64(len(gate.Criteria)-1)
	for label := range gate.Criteria {
		out[label] = rest
	}
	out[top] = p
	return out
}

func TestJudgeVerdicts(t *testing.T) {
	tests := []struct {
		name       string
		def        gateDef
		label      string
		confidence float64
		want       string
	}{
		{"closing a todo above the high threshold acts", gateTodoState, "done", 0.9, verdictAct},
		{"closing a todo at the high threshold acts", gateTodoState, "done", 0.8, verdictAct},
		{"closing a todo between low and high escalates", gateTodoState, "done", 0.75, verdictEscalate},
		{"a rewrite needs the high threshold too", gateTodoState, "stale", 0.7, verdictEscalate},
		{"a low-stakes label acts at the low threshold", gateTodoState, "open", 0.65, verdictAct},
		{"below the floor escalates", gateTodoState, "open", 0.55, verdictEscalate},
		{"unknown escalates at full confidence", gateTodoState, "unknown", 0.99, verdictEscalate},
		{"session activity acts at the low threshold", gateSession, "implementing", 0.62, verdictAct},
		{"session unknown escalates", gateSession, "unknown", 0.95, verdictEscalate},
		{"a done reply acts above high", gateReply, "done", 0.85, verdictAct},
		{"a done reply below high escalates", gateReply, "done", 0.7, verdictEscalate},
		{"a not_done reply acts at low", gateReply, "not_done", 0.65, verdictAct},
		{"an unclear reply escalates", gateReply, "unclear", 0.99, verdictEscalate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeGate(t, serve(200, answerBody(tt.def.Name, tt.label, probs(tt.def, tt.label, 0.9), tt.confidence)))
			g, _ := testGate(t, f.srv.URL)
			j := g.Judge(t.Context(), "todo", "t1", tt.def, map[string]string{"id": "t1"})
			assert.Equal(t, tt.want, j.Verdict, j.Reason)
			assert.Equal(t, tt.label, j.Label)
			assert.InDelta(t, tt.confidence, j.Confidence, 1e-9)
			assert.Equal(t, decidedByGate, j.DecidedBy)
			assert.Equal(t, 1, f.requests())
		})
	}
}

func TestJudgeReplaysSyntheticResponses(t *testing.T) {
	tests := []struct {
		file  string
		def   gateDef
		label string
		want  string
	}{
		{"todo_state_done.json", gateTodoState, "done", verdictAct},
		{"session_activity_implementing.json", gateSession, "implementing", verdictAct},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join("testdata", "gate", tt.file))
			require.NoError(t, err)
			f := newFakeGate(t, serve(200, string(body)))
			g, _ := testGate(t, f.srv.URL)
			j := g.Judge(t.Context(), "todo", "t1", tt.def, "state")
			assert.Equal(t, tt.label, j.Label)
			assert.Equal(t, tt.want, j.Verdict)
		})
	}
}

func TestJudgeEscalatesOnInvalidOrFailedCalls(t *testing.T) {
	d := gateTodoState
	good := probs(d, "done", 0.9)
	with := func(mutate func(map[string]float64)) map[string]float64 {
		m := map[string]float64{}
		for k, v := range good {
			m[k] = v
		}
		mutate(m)
		return m
	}
	tests := []struct {
		name         string
		reply        func(int, http.ResponseWriter)
		wantRequests int
		wantReason   string
	}{
		{"malformed JSON", serve(200, `{"answers": [`), 1, "not valid JSON"},
		{"no answer for the gate", serve(200, `{"model":"m","answers":{},"usage":{}}`), 1, "no answer"},
		{"wrong answer type", serve(200, `{"answers":{"todo_state":{"type":"noul","noul":0.9}}}`), 1, "want choice"},
		{"choice outside the labels", serve(200, answerBody(d.Name, "finished", good, 0.9)), 1, "not an offered label"},
		{"a label missing from probabilities", serve(200, answerBody(d.Name, "done", with(func(m map[string]float64) {
			delete(m, "stale")
			m["done"] += 0.1
		}), 0.9)), 1, "do not cover"},
		{"an extra label in probabilities", serve(200, answerBody(d.Name, "done", with(func(m map[string]float64) {
			m["extra"] = 0
		}), 0.9)), 1, "do not cover"},
		{"probabilities sum to 0.98", serve(200, answerBody(d.Name, "done", with(func(m map[string]float64) {
			m["open"] -= 0.02
		}), 0.9)), 1, "sum to"},
		{"a negative probability", serve(200, answerBody(d.Name, "done", with(func(m map[string]float64) {
			m["open"] = -0.1
			m["done"] += 0.14
		}), 0.9)), 1, "outside 0 to 1"},
		{"confidence above 1", serve(200, answerBody(d.Name, "done", good, 1.4)), 1, "confidence"},
		{"confidence missing", serve(200, strings.Replace(answerBody(d.Name, "done", good, 0.9), `,"confidence":0.9`, "", 1)), 1, "confidence"},
		{"401 is not retried", serve(401, `{"error":"bad key"}`), 1, "HTTP 401"},
		{"422 is not retried", serve(422, `{"error":"bad body"}`), 1, "HTTP 422"},
		{"500 is not retried", serve(500, ""), 1, "HTTP 500"},
		{"429 forever exhausts the retries", serve(429, ""), 5, "HTTP 429"},
		{"529 forever exhausts the retries", serve(529, ""), 5, "HTTP 529"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeGate(t, tt.reply)
			g, _ := testGate(t, f.srv.URL)
			j := g.Judge(t.Context(), "todo", "t1", d, "state")
			assert.Equal(t, verdictEscalate, j.Verdict)
			assert.Equal(t, decidedByError, j.DecidedBy)
			assert.Empty(t, j.Label)
			assert.Contains(t, j.Reason, tt.wantReason)
			assert.NotContains(t, j.Reason, keySentinel)
			assert.Equal(t, tt.wantRequests, f.requests())
		})
	}

	t.Run("a closed server escalates", func(t *testing.T) {
		f := newFakeGate(t, serve(200, ""))
		g, _ := testGate(t, f.srv.URL)
		f.srv.Close()
		j := g.Judge(t.Context(), "todo", "t1", d, "state")
		assert.Equal(t, verdictEscalate, j.Verdict)
		assert.Contains(t, j.Reason, "gate unreachable")
		assert.NotContains(t, j.Reason, keySentinel)
	})

	t.Run("a cancelled context escalates", func(t *testing.T) {
		f := newFakeGate(t, serve(200, answerBody(d.Name, "done", good, 0.9)))
		g, _ := testGate(t, f.srv.URL)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		j := g.Judge(ctx, "todo", "t1", d, "state")
		assert.Equal(t, verdictEscalate, j.Verdict)
	})
}

func TestJudgeRetriesThrottledCalls(t *testing.T) {
	d := gateTodoState
	ok := answerBody(d.Name, "done", probs(d, "done", 0.9), 0.9)
	tests := []struct {
		name         string
		throttle     int
		status       int
		retryAfter   string
		wantRequests int
	}{
		{"one 429 then success", 1, 429, "0", 2},
		{"two 529 then success", 2, 529, "0", 3},
		{"four 429 then success on the last retry", 4, 429, "0", 5},
		{"a day-long Retry-After on a 429 is capped at the maximum wait", 1, 429, "86400", 2},
		{"a day-long Retry-After on a 529 is capped at the maximum wait", 1, 529, "86400", 2},
		{"an overflowing Retry-After on a 429 is capped at the maximum wait", 1, 429, "9223372036854775807", 2},
		{"an overflowing Retry-After on a 529 is capped at the maximum wait", 1, 529, "9223372036854775807", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeGate(t, func(n int, w http.ResponseWriter) {
				if n <= tt.throttle {
					w.Header().Set("Retry-After", tt.retryAfter)
					w.WriteHeader(tt.status)
					return
				}
				_, _ = io.WriteString(w, ok)
			})
			g, _ := testGate(t, f.srv.URL)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			j := g.Judge(ctx, "todo", "t1", d, "state")
			assert.Equal(t, verdictAct, j.Verdict, j.Reason)
			assert.Equal(t, tt.wantRequests, f.requests())
		})
	}
}

func TestRequestShape(t *testing.T) {
	for _, d := range []gateDef{gateTodoState, gateSession, gateReply} {
		t.Run(d.Name, func(t *testing.T) {
			label := d.Unknown
			f := newFakeGate(t, serve(200, answerBody(d.Name, label, probs(d, label, 0.9), 0.9)))
			g, _ := testGate(t, f.srv.URL)
			state := map[string]string{"text": "synthetic state"}
			a, hash, err := g.ask(t.Context(), state, d)
			require.NoError(t, err)
			assert.Len(t, hash, 16)
			assert.Equal(t, answer{Label: label, Confidence: 0.9}, a)
			require.Equal(t, 1, f.requests())

			h := f.headers[0]
			assert.Equal(t, "Bearer "+keySentinel, h.Get("Authorization"))
			assert.Equal(t, "application/json", h.Get("Content-Type"))
			var sent struct {
				State     map[string]string `json:"state"`
				Model     string            `json:"model"`
				Questions map[string]struct {
					Type     string            `json:"type"`
					Criteria map[string]string `json:"criteria"`
				} `json:"questions"`
			}
			require.NoError(t, json.Unmarshal(f.bodies[0], &sent))
			assert.Equal(t, state, sent.State)
			assert.Equal(t, "jev-latest", sent.Model)
			require.Len(t, sent.Questions, 1)
			q := sent.Questions[d.Name]
			assert.Equal(t, "choice", q.Type)
			assert.Contains(t, q.Criteria, d.Unknown, "the label set always includes the unknown label")
			assert.Len(t, q.Criteria, len(d.Criteria))
			assert.NotContains(t, string(f.bodies[0]), keySentinel)
		})
	}
}

func TestReadAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		keyring func(t *testing.T)
		want    string
		wantErr string
	}{
		{"the environment wins", keySentinel, func(t *testing.T) {
			require.NoError(t, keyring.Set(keyringService, keyringUser, "from-keyring"))
		}, keySentinel, ""},
		{"the keyring is the fallback", "", func(t *testing.T) {
			require.NoError(t, keyring.Set(keyringService, keyringUser, "from-keyring"))
		}, "from-keyring", ""},
		{"a keyring miss is an error", "", func(*testing.T) {}, "", "no API key"},
		{"an empty keyring value is an error", "", func(t *testing.T) {
			require.NoError(t, keyring.Set(keyringService, keyringUser, ""))
		}, "", "no API key"},
		{"a keyring failure is an error", "", func(*testing.T) { keyring.MockInitWithError(fmt.Errorf("no session bus")) }, "", "no session bus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyring.MockInit()
			t.Setenv(keyEnv, tt.env)
			tt.keyring(t)
			key, err := readAPIKey()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, key)
			_, set := os.LookupEnv(keyEnv)
			assert.False(t, set, "the key must be out of the environment before any child process starts")
		})
	}
}

func TestNoKeyEscalatesEveryGate(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T)
		direct bool // build the gate with an empty key and no error, as a bug would
	}{
		{"a keyring miss with no environment key", func(*testing.T) {}, false},
		{"an empty keyring value", func(t *testing.T) { require.NoError(t, keyring.Set(keyringService, keyringUser, "")) }, false},
		{"a keyring failure", func(*testing.T) { keyring.MockInitWithError(fmt.Errorf("no session bus")) }, false},
		{"an empty key with no error", func(*testing.T) {}, true},
	}
	for _, tt := range tests {
		for _, d := range []gateDef{gateTodoState, gateSession, gateReply} {
			t.Run(tt.name+"/"+d.Name, func(t *testing.T) {
				keyring.MockInit()
				t.Setenv(keyEnv, "")
				tt.setup(t)
				key, keyErr := readAPIKey()
				if tt.direct {
					key, keyErr = "", nil
				} else {
					require.Error(t, keyErr)
				}
				f := newFakeGate(t, serve(200, ""))
				logger, file, err := openGateLog(filepath.Join(t.TempDir(), "gate.log"))
				require.NoError(t, err)
				t.Cleanup(func() { _ = file.Close() })
				g := newGate(GateConfig{URL: f.srv.URL}, testThresholds, key, keyErr, logger, testNow)

				j := g.Judge(t.Context(), "todo", "t1", d, "state")
				assert.Equal(t, verdictEscalate, j.Verdict)
				assert.Equal(t, decidedByError, j.DecidedBy)
				assert.Contains(t, j.Reason, "no API key")
				assert.Empty(t, j.Hash)
				assert.Zero(t, f.requests(), "no request may be sent without a key")
			})
		}
	}
}

func TestGateCallLog(t *testing.T) {
	d := gateTodoState
	good := answerBody(d.Name, "done", probs(d, "done", 0.9), 0.9)
	tests := []struct {
		name      string
		reply     func(int, http.ResponseWriter)
		wantLines int
		check     func(t *testing.T, entry map[string]any, j Judgment)
	}{
		{"a successful call logs the gate, subject, hash, label, confidence and verdict", serve(200, good), 1,
			func(t *testing.T, e map[string]any, j Judgment) {
				assert.Equal(t, "todo_state", e["gate"])
				assert.Equal(t, "todo", e["subject"])
				assert.Equal(t, "t1", e["id"])
				assert.Equal(t, j.Hash, e["input_hash"])
				assert.Equal(t, "done", e["label"])
				assert.Equal(t, 0.9, e["confidence"])
				assert.Equal(t, verdictAct, e["verdict"])
				assert.Equal(t, "cto todo_state done 0.90 act "+j.Hash[:8]+" 2026-10-08", j.Annotation)
			}},
		{"a failed call logs its reason and escalates", serve(500, ""), 1,
			func(t *testing.T, e map[string]any, j Judgment) {
				assert.Equal(t, verdictEscalate, e["verdict"])
				assert.Equal(t, "gate returned HTTP 500", e["reason"])
				assert.Equal(t, j.Hash, e["input_hash"])
				assert.Empty(t, j.Annotation)
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeGate(t, tt.reply)
			g, logPath := testGate(t, f.srv.URL)
			state := map[string]string{"text": "private todo text"}
			j := g.Judge(t.Context(), "todo", "t1", d, state)

			data, err := os.ReadFile(logPath)
			require.NoError(t, err)
			lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
			require.Len(t, lines, tt.wantLines)
			var entry map[string]any
			require.NoError(t, json.Unmarshal(lines[0], &entry))
			tt.check(t, entry, j)
			assert.NotContains(t, string(data), "private todo text", "the log holds a hash, never the state")
			assert.NotContains(t, string(data), keySentinel)
		})
	}
}
