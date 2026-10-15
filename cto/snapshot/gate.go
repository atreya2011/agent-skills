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
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/zalando/go-keyring"
)

const (
	gateModel      = "jev-latest"
	keyEnv         = "TYPESAFE_API_KEY"
	keyringService = "typesafe"
	keyringUser    = "api"

	verdictAct      = "act"
	verdictEscalate = "escalate"

	decidedByRule  = "rule"
	decidedByGate  = "gate"
	decidedByError = "error"

	statusOverloaded = 529

	probabilityTolerance = 0.01
)

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
			"the linked pull requests, issues and commits, the tabs of its project, linked wiki pages and matching handoff files. " +
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
	gateTodoOwner = gateDef{
		Name:         "todo_owner",
		Instructions: "Who owns the remaining work in this todo? Judge only from the todo and its evidence. Use ownership_policy in the state for the CTO and CoS responsibility split. The user owns personal decisions, approvals, physical actions and work requiring their identity. Choose unknown when the work requires different owners or the evidence does not decide. A technical subject alone does not make personal approval or physical work the CTO responsibility.",
		Criteria: map[string]string{
			"user":    "The user must personally decide, approve or perform the remaining work.",
			"cto":     "The CTO can carry out the remaining work under ownership_policy.",
			"cos":     "The CoS can carry out the remaining work under ownership_policy.",
			"unknown": "The evidence is insufficient or the todo needs splitting between owners.",
		},
		Unknown: "unknown",
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
		Instructions: "The CTO asked an agent whether its tab or tabs are still needed. What does the reply in the state say?",
		Criteria: map[string]string{
			"done":     "The reply explicitly says its work is finished and its tab or tabs can be closed.",
			"not_done": "The reply says work continues or its tab or tabs are still needed.",
			"unclear":  "The reply does not clearly say either.",
		},
		Unknown: "unclear",
		High:    []string{"done"},
	}
)

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

type Gate struct {
	url    string
	key    string
	keyErr error
	th     Thresholds
	client *retryablehttp.Client
	log    *slog.Logger
	today  string
}

var errNoKey = fmt.Errorf("no API key: %s is unset and the keyring has no entry for service %s, user %s",
	keyEnv, keyringService, keyringUser)

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
	case errors.Is(err, keyring.ErrNotFound) || (err == nil && key == ""):
		return "", errNoKey
	case err != nil:
		return "", fmt.Errorf("no API key: %s is unset and the keyring failed: %w", keyEnv, err)
	}
	return key, nil
}

func newGate(cfg GateConfig, th Thresholds, key string, keyErr error, log *slog.Logger, now time.Time) *Gate {
	if key == "" && keyErr == nil {
		keyErr = errNoKey
	}
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
		url: cfg.URL, key: key, keyErr: keyErr, th: th,
		client: c, log: log, today: now.Format(time.DateOnly),
	}
}

func retryOnThrottle(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if err != nil {
		return false, nil
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == statusOverloaded, nil
}

func jitteredBackoff(minWait, maxWait time.Duration, attempt int, resp *http.Response) time.Duration {
	wait := retryablehttp.DefaultBackoff(minWait, maxWait, attempt, resp)
	if resp != nil && resp.StatusCode == statusOverloaded {
		if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && secs >= 0 {
			wait = time.Duration(secs) * time.Second
		}
	}
	if wait <= 0 || wait > maxWait {
		wait = maxWait
	}
	return min(wait+rand.N(wait/4+1), maxWait)
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

func (g *Gate) ask(ctx context.Context, state any, d gateDef) (answer, string, error) {
	if g.key == "" {
		return answer{}, "", g.keyErr
	}
	req := request{
		State: state, Model: gateModel,
		Questions: map[string]question{d.Name: {Type: "choice", Instructions: d.Instructions, Criteria: d.Criteria}},
	}
	body, err := json.Marshal(req, json.Deterministic(true))
	if err != nil {
		return answer{}, "", fmt.Errorf("encode gate request: %w", err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])[:16]

	hreq, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, g.url, body)
	if err != nil {
		return answer{}, hash, fmt.Errorf("build gate request: %w", err)
	}
	hreq.Header.Set("Authorization", "Bearer "+g.key)
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(hreq)
	if err != nil {
		return answer{}, hash, fmt.Errorf("gate unreachable: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return answer{}, hash, fmt.Errorf("read gate response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return answer{}, hash, &gateHTTPError{Status: resp.StatusCode, Detail: string(data)}
	}
	var parsed response
	if err := json.Unmarshal(data, &parsed); err != nil {
		return answer{}, hash, fmt.Errorf("gate response is not valid JSON: %w", err)
	}
	raw, ok := parsed.Answers[d.Name]
	if !ok {
		return answer{}, hash, fmt.Errorf("gate response has no answer for %s", d.Name)
	}
	a, err := validateAnswer(d, raw)
	if err != nil {
		return answer{}, hash, fmt.Errorf("gate answer for %s: %w", d.Name, err)
	}
	return a, hash, nil
}

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

func (g *Gate) verdict(d gateDef, label string, confidence float64) (string, string) {
	need := g.th.Low
	if d.Name == gateTodoOwner.Name {
		need = g.th.Floor
	}
	if slices.Contains(d.High, label) {
		need = g.th.High
	}
	need = max(need, g.th.Floor)
	switch {
	case d.Name == gateTodoOwner.Name && label == "user":
		return verdictEscalate, "the user owns this todo"
	case label == d.Unknown:
		return verdictEscalate, "the label is " + label
	case confidence < need:
		return verdictEscalate, fmt.Sprintf("confidence %.2f is below %.2f", confidence, need)
	}
	return verdictAct, ""
}

func (g *Gate) Judge(ctx context.Context, subject, id string, d gateDef, state any) Judgment {
	j := Judgment{Subject: subject, ID: id, Gate: d.Name, Verdict: verdictEscalate, DecidedBy: decidedByError}
	a, hash, err := g.ask(ctx, state, d)
	j.Hash = hash
	if err != nil {
		j.Reason = err.Error()
		if state, ok := state.(todoState); ok && inputLimitError(err) {
			j.Reason = g.limitPage(ctx, state, d, err)
		}
	} else {
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

type gateHTTPError struct {
	Status int
	Detail string
}

func (e *gateHTTPError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("gate returned HTTP %d", e.Status)
	}
	return fmt.Sprintf("gate returned HTTP %d: %s", e.Status, e.Detail)
}

func inputLimitError(err error) bool {
	e, ok := errors.AsType[*gateHTTPError](err)
	if !ok || (e.Status != 422 && e.Status != 413 && e.Status != 400) {
		return false
	}
	detail := strings.ToLower(e.Detail)
	return e.Status == 413 || strings.Contains(detail, "token") || strings.Contains(detail, "context length") || strings.Contains(detail, "too large") || strings.Contains(detail, "maximum context")
}

func (g *Gate) limitPage(ctx context.Context, state todoState, d gateDef, original error) string {
	wiki, handoffs := state.WikiPages, state.Handoffs
	state.WikiPages, state.Handoffs = nil, nil
	_, _, err := g.ask(ctx, state, d)
	if inputLimitError(err) {
		return "TypeSafe input limit exceeded by todo facts before any wiki or handoff page; no text truncated"
	}
	if err != nil {
		return original.Error() + "; cannot identify the page: " + err.Error()
	}
	for _, group := range []struct {
		pages []contextPage
		wiki  bool
	}{{wiki, true}, {handoffs, false}} {
		for _, page := range group.pages {
			if group.wiki {
				state.WikiPages = append(state.WikiPages, page)
			} else {
				state.Handoffs = append(state.Handoffs, page)
			}
			_, _, err := g.ask(ctx, state, d)
			if inputLimitError(err) {
				return "TypeSafe input limit exceeded when adding page " + page.Path + "; no text truncated"
			}
			if err != nil {
				return original.Error() + "; cannot identify the page after " + page.Path + ": " + err.Error()
			}
		}
	}
	return original.Error() + "; no text truncated; page limit could not be reproduced"
}
