package main

import (
	"context"
	"slices"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// maxParallel bounds concurrent source commands and gate calls.
const maxParallel = 8

// parallel runs fn for every item with at most maxParallel in flight. fn
// writes its result to its own slot, so no result needs a lock.
func parallel[T any](items []T, fn func(i int, item T)) {
	var g errgroup.Group
	g.SetLimit(maxParallel)
	for i, item := range items {
		g.Go(func() error {
			fn(i, item)
			return nil
		})
	}
	_ = g.Wait()
}

// tabBrief is the part of a tab a gate sees.
type tabBrief struct {
	Label string `json:"label"`
	Agent string `json:"agent,omitempty"`
	State string `json:"state"`
	AgeS  *int   `json:"age_s"`
}

// todoState is what the todo_state gate sees about one todo.
type todoState struct {
	ID          string     `json:"id"`
	Today       string     `json:"today"`
	Description string     `json:"description"`
	Domain      string     `json:"domain,omitempty"`
	Due         string     `json:"due,omitempty"`
	Annotations []string   `json:"annotations"`
	Links       []Link     `json:"links"`
	ProjectTabs []tabBrief `json:"project_tabs"`
	Transcript  string     `json:"transcript_excerpt,omitempty"`
}

// sessionState is what the session_activity gate sees about one session.
type sessionState struct {
	ID       string `json:"id"`
	Profile  string `json:"profile"`
	Project  string `json:"project"`
	AgeS     int    `json:"age_s"`
	TabState string `json:"tab_state,omitempty"`
	Excerpt  string `json:"excerpt"`
}

// gateAnnotations drops the annotations the CTO wrote itself: a pending
// question ("ask:") and a recorded verdict ("cto "). The gate must judge the
// todo from the facts, not from its own earlier output.
func gateAnnotations(all []string) []string {
	return slices.DeleteFunc(slices.Clone(all), func(a string) bool {
		return strings.HasPrefix(a, "ask:") || strings.HasPrefix(a, "cto ")
	})
}

// judgeAll returns one judgment per judged todo and per session, in the order
// of the projects. An untagged todo escalates, because it has no assignee yet.
// Rules decide first. A todo with an unreadable link source
// escalates without a gate call. Everything else goes to a gate, in parallel.
// excerptFor returns the transcript excerpt of a session a todo cites.
func judgeAll(ctx context.Context, gate *Gate, projects []*Project, now time.Time, excerptFor func(id string) string) []Judgment {
	today := now.Format(time.DateOnly)
	var jobs []func() Judgment
	fixed := func(j Judgment) { jobs = append(jobs, func() Judgment { return j }) }
	for _, p := range projects {
		var tabs []tabBrief
		stateOf := map[string]string{}
		for _, t := range p.Tabs {
			tabs = append(tabs, tabBrief{Label: t.Label, Agent: t.Agent, State: t.State, AgeS: t.AgeS})
			stateOf[t.ID] = t.State
		}
		for _, t := range p.Todos {
			if !judged(t) {
				continue
			}
			if t.Assignee == "" {
				fixed(Judgment{Subject: "todo", ID: t.UUID, Verdict: verdictEscalate, DecidedBy: decidedByError, Reason: "no assignee tag"})
				continue
			}
			if j, ok := ruleJudgment(t, today); ok {
				fixed(j)
				continue
			}
			if slices.ContainsFunc(t.Links, func(l Link) bool { return l.Unreadable }) {
				fixed(Judgment{Subject: "todo", ID: t.UUID, Gate: gateTodoState.Name, Verdict: verdictEscalate,
					DecidedBy: decidedByError, Reason: "a source this todo cites is unreadable"})
				continue
			}
			state := todoState{
				ID: t.UUID, Today: today, Description: t.Description, Domain: t.Domain, Due: t.Due,
				Annotations: gateAnnotations(t.Annotations), Links: t.Links, ProjectTabs: tabs,
			}
			if t.SessionID != "" {
				state.Transcript = excerptFor(t.SessionID)
			}
			jobs = append(jobs, func() Judgment { return gate.Judge(ctx, "todo", t.UUID, gateTodoState, state) })
		}
		for _, s := range p.Sessions {
			if s.Excerpt == "" {
				fixed(Judgment{Subject: "session", ID: s.ID, Gate: gateSession.Name, Verdict: verdictEscalate,
					DecidedBy: decidedByError, Reason: "the transcript tail has no readable text"})
				continue
			}
			state := sessionState{ID: s.ID, Profile: s.Profile, Project: p.Label, AgeS: s.AgeS, TabState: stateOf[s.TabID], Excerpt: s.Excerpt}
			jobs = append(jobs, func() Judgment { return gate.Judge(ctx, "session", s.ID, gateSession, state) })
		}
	}
	out := make([]Judgment, len(jobs))
	parallel(jobs, func(i int, job func() Judgment) { out[i] = job() })
	return out
}
