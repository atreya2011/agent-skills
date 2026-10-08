package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/zalando/go-keyring"
)

const (
	keyEnv         = "TYPESAFE_API_KEY"
	keyringService = "typesafe"
	keyringUser    = "api"

	verdictAct      = "act"
	verdictEscalate = "escalate"

	decidedByRule  = "rule"
	decidedByGate  = "gate"
	decidedByError = "error"

	// statusOverloaded is the status TypeSafe returns when it is overloaded.
	statusOverloaded = 529
	// probabilityTolerance is how far the probabilities of an answer may be
	// from summing to 1.
	probabilityTolerance = 0.01
)

// gateDef is one fixed gate: the question it asks, the labels it offers and
// the labels whose act needs the high threshold because the act is hard to
// undo. Unknown is the label that always escalates.
type gateDef struct {
	Name         string
	Instructions string
	Criteria     map[string]string
	Unknown      string
	High         []string
}

var (
	gateTodoState = gateDef{
		Name: "todo_state",
		Instructions: "A todo is tracked in Taskwarrior. Compare it with the facts in the state: its text and annotations, " +
			"the linked pull requests, issues and commits, the tabs of its project and the transcript excerpt. " +
			"Judge only from the state. Which label describes the todo now?",
		Criteria: map[string]string{
			"done":    "The facts show the work is finished.",
			"open":    "The work is still to do or in progress, and the todo text still matches it.",
			"stale":   "The todo text no longer matches the facts: scope, links or status changed and the text needs a rewrite.",
			"unknown": "The facts do not decide between the other labels.",
		},
		Unknown: "unknown",
		High:    []string{"done", "stale"},
	}
	gateSession = gateDef{
		Name:         "session_activity",
		Instructions: "What is this agent session doing? Judge only from the end of its transcript in the state.",
		Criteria: map[string]string{
			"implementing":  "Writing or changing code, tests or documents.",
			"reviewing":     "Reviewing code, a diff or a pull request.",
			"investigating": "Reading, searching or diagnosing without changing files.",
			"blocked":       "Waiting for a user decision, an approval or missing input.",
			"chatting":      "Talking with the user with no task in progress.",
			"unknown":       "The excerpt does not show the activity.",
		},
		Unknown: "unknown",
	}
	gateReply = gateDef{
		Name:         "orchestrator_reply",
		Instructions: "The CTO asked an orchestrator whether its tabs are still needed. What does the reply in the state say?",
		Criteria: map[string]string{
			"done":     "The reply explicitly says its work is finished and its tabs can be closed.",
			"not_done": "The reply says work continues or the tabs are still needed.",
			"unclear":  "The reply does not clearly say either.",
		},
		Unknown: "unclear",
		High:    []string{"done"},
	}
)

// Judgment is the outcome for one todo, session or reply: the gate or rule
// that decided, its label and confidence, and whether the CTO may act on it.
type Judgment struct {
	Subject    string  `json:"subject"`
	ID         string  `json:"id,omitempty"`
	Gate       string  `json:"gate,omitempty"`
	Label      string  `json:"label,omitempty"`
	Confidence float64 `json:"confidence"`
	Verdict    string  `json:"verdict"`
	DecidedBy  string  `json:"decided_by"`
	Rule       string  `json:"rule,omitempty"`
	Reason     string  `json:"reason,omitempty"`
	Hash       string  `json:"hash,omitempty"`
	Annotation string  `json:"annotation,omitempty"`
}

// Gate calls the TypeSafe API. It holds the API key in an unexported field so
// the key only ever reaches the Authorization header.
type Gate struct {
	url    string
	model  string
	key    string
	keyErr error
	th     Thresholds
	client *retryablehttp.Client
	log    *slog.Logger
	today  string
}

// readAPIKey returns the API key from the environment, else from the OS
// keyring. It clears the environment variable first, so no child process the
// tool starts can inherit the key.
func readAPIKey() (string, error) {
	key := os.Getenv(keyEnv)
	if err := os.Unsetenv(keyEnv); err != nil {
		return "", fmt.Errorf("no API key: %w", err)
	}
	if key != "" {
		return key, nil
	}
	key, err := keyring.Get(keyringService, keyringUser)
	switch {
	case errors.Is(err, keyring.ErrNotFound):
		return "", fmt.Errorf("no API key: %s is unset and the keyring has no entry for service %s, user %s",
			keyEnv, keyringService, keyringUser)
	case err != nil:
		return "", fmt.Errorf("no API key: %s is unset and the keyring failed: %w", keyEnv, err)
	}
	return key, nil
}

// newGate builds a gate. A non-nil keyErr means no key was found, and every
// judgment then escalates without a request.
func newGate(cfg GateConfig, th Thresholds, key string, keyErr error, log *slog.Logger, now time.Time) *Gate {
	c := retryablehttp.NewClient()
	c.Logger = nil
	c.RetryMax = 4
	c.RetryWaitMin = time.Second
	c.RetryWaitMax = 16 * time.Second
	c.HTTPClient.Timeout = 30 * time.Second
	c.CheckRetry = retryOnThrottle
	c.Backoff = jitteredBackoff
	c.ErrorHandler = retryablehttp.PassthroughErrorHandler
	return &Gate{
		url: cfg.URL, model: cfg.Model, key: key, keyErr: keyErr, th: th,
		client: c, log: log, today: now.Format(time.DateOnly),
	}
}

// retryOnThrottle retries only the statuses the API documents as transient,
// 429 and 529. A transport error is returned to the caller, which escalates.
func retryOnThrottle(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		return false, nil
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == statusOverloaded, nil
}

// jitteredBackoff waits for the Retry-After seconds the server names, else
// for an exponential delay, plus up to a quarter more so that parallel calls
// do not retry in step. The default backoff reads Retry-After for 429 only.
func jitteredBackoff(minWait, maxWait time.Duration, attempt int, resp *http.Response) time.Duration {
	wait := retryablehttp.DefaultBackoff(minWait, maxWait, attempt, resp)
	if resp != nil && resp.StatusCode == statusOverloaded {
		if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs >= 0 {
			wait = min(time.Duration(secs)*time.Second, maxWait)
		}
	}
	return wait + rand.N(wait/4+1)
}

type question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type request struct {
	State     any                 `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]question `json:"questions"`
}

type rawAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    *float64           `json:"confidence"`
}

type response struct {
	Answers map[string]rawAnswer `json:"answers"`
}

type answer struct {
	Label      string
	Confidence float64
}

// ask sends one request that carries the state and one question per gate, and
// returns the validated answers by gate name with the hash of the request. A
// response that does not match the documented shape and the offered labels is
// an error, never a partial answer.
func (g *Gate) ask(ctx context.Context, state any, defs ...gateDef) (map[string]answer, string, error) {
	if g.key == "" {
		return nil, "", g.keyErr
	}
	req := request{State: state, Model: g.model, Questions: map[string]question{}}
	for _, d := range defs {
		req.Questions[d.Name] = question{Type: "choice", Instructions: d.Instructions, Criteria: d.Criteria}
	}
	body, err := json.Marshal(req, json.Deterministic(true))
	if err != nil {
		return nil, "", fmt.Errorf("encode gate request: %w", err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])[:16]

	hreq, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, g.url, body)
	if err != nil {
		return nil, hash, fmt.Errorf("build gate request: %w", err)
	}
	hreq.Header.Set("Authorization", "Bearer "+g.key)
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(hreq)
	if err != nil {
		return nil, hash, fmt.Errorf("gate unreachable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, hash, fmt.Errorf("read gate response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, hash, fmt.Errorf("gate returned HTTP %d", resp.StatusCode)
	}
	var parsed response
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, hash, fmt.Errorf("gate response is not valid JSON: %w", err)
	}
	out := map[string]answer{}
	for _, d := range defs {
		raw, ok := parsed.Answers[d.Name]
		if !ok {
			return nil, hash, fmt.Errorf("gate response has no answer for %s", d.Name)
		}
		a, err := validateAnswer(d, raw)
		if err != nil {
			return nil, hash, fmt.Errorf("gate answer for %s: %w", d.Name, err)
		}
		out[d.Name] = a
	}
	return out, hash, nil
}

// validateAnswer accepts a choice answer only when its probabilities cover
// exactly the offered labels and sum to 1 within the tolerance.
func validateAnswer(d gateDef, raw rawAnswer) (answer, error) {
	if raw.Type != "choice" {
		return answer{}, fmt.Errorf("type is %q, want choice", raw.Type)
	}
	if _, ok := d.Criteria[raw.Choice]; !ok {
		return answer{}, fmt.Errorf("choice %q is not an offered label", raw.Choice)
	}
	if len(raw.Probabilities) != len(d.Criteria) {
		return answer{}, errors.New("probabilities do not cover exactly the offered labels")
	}
	sum := 0.0
	for label := range d.Criteria {
		p, ok := raw.Probabilities[label]
		if !ok {
			return answer{}, fmt.Errorf("probabilities lack label %q", label)
		}
		if p < 0 || p > 1 || math.IsNaN(p) {
			return answer{}, fmt.Errorf("probability %v of %q is outside 0 to 1", p, label)
		}
		sum += p
	}
	if math.Abs(sum-1) > probabilityTolerance+1e-9 {
		return answer{}, fmt.Errorf("probabilities sum to %.3f, not 1", sum)
	}
	if raw.Confidence == nil || *raw.Confidence < 0 || *raw.Confidence > 1 {
		return answer{}, errors.New("confidence is missing or outside 0 to 1")
	}
	return answer{Label: raw.Choice, Confidence: *raw.Confidence}, nil
}

// verdict decides whether the CTO may act on a label. It escalates the gate's
// unknown label and any confidence below the floor or the label's threshold.
func (g *Gate) verdict(d gateDef, label string, confidence float64) (string, string) {
	need := g.th.Low
	if slices.Contains(d.High, label) {
		need = g.th.High
	}
	need = max(need, g.th.Floor)
	switch {
	case label == d.Unknown:
		return verdictEscalate, "the label is " + label
	case confidence < need:
		return verdictEscalate, fmt.Sprintf("confidence %.2f is below %.2f", confidence, need)
	}
	return verdictAct, ""
}

// Judge asks one gate about one subject and logs the call. Any error leaves
// the judgment at escalate; it never acts on a failed or invalid call.
func (g *Gate) Judge(ctx context.Context, subject, id string, d gateDef, state any) Judgment {
	j := Judgment{Subject: subject, ID: id, Gate: d.Name, Verdict: verdictEscalate, DecidedBy: decidedByError}
	answers, hash, err := g.ask(ctx, state, d)
	j.Hash = hash
	if err != nil {
		j.Reason = err.Error()
	} else {
		a := answers[d.Name]
		j.Label, j.Confidence, j.DecidedBy = a.Label, a.Confidence, decidedByGate
		j.Verdict, j.Reason = g.verdict(d, a.Label, a.Confidence)
		j.Annotation = fmt.Sprintf("cto %s %s %.2f %s %s %s", d.Name, j.Label, j.Confidence, j.Verdict, hash[:8], g.today)
	}
	if hash != "" {
		g.log.Info("gate call", "gate", d.Name, "subject", subject, "id", id, "input_hash", hash,
			"label", j.Label, "confidence", j.Confidence, "verdict", j.Verdict, "reason", j.Reason)
	}
	return j
}

// openGateLog opens the gate-call log for appending and returns a JSON logger
// that writes to it. The caller closes the file.
func openGateLog(path string) (*slog.Logger, *os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, nil, fmt.Errorf("gate log: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("gate log: %w", err)
	}
	return slog.New(slog.NewJSONHandler(f, nil)), f, nil
}
