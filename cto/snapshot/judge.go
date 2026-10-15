package main

import (
	"context"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
)

const maxParallel = 8

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

type tabBrief struct {
	Label string `json:"label"`
	Agent string `json:"agent,omitempty"`
	State string `json:"state"`
	AgeS  *int   `json:"age_s"`
}

type todoState struct {
	ID          string        `json:"id"`
	Today       string        `json:"today"`
	Description string        `json:"description"`
	Domain      string        `json:"domain,omitempty"`
	Due         time.Time     `json:"due,omitzero"`
	Annotations []string      `json:"annotations"`
	Links       []Link        `json:"links"`
	ProjectTabs []tabBrief    `json:"project_tabs"`
	WikiPages   []contextPage `json:"wiki_pages,omitempty"`
	OwnerPolicy string        `json:"ownership_policy,omitempty"`
	Handoffs    []contextPage `json:"handoffs,omitempty"`
}

type sessionState struct {
	ID       string `json:"id"`
	Profile  string `json:"profile"`
	Project  string `json:"project"`
	AgeS     int    `json:"age_s"`
	TabState string `json:"tab_state,omitempty"`
	Excerpt  string `json:"excerpt"`
}

func gateAnnotations(all []string) []string {
	return slices.DeleteFunc(slices.Clone(all), ownAnnotation)
}

func judgeAll(ctx context.Context, gate *Gate, projects []*Project, now time.Time, cfg Config) []Judgment {
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
			if t.Assignee == "" && (!cfg.Gate.TodoContextEnabled || t.Status != "pending") {
				fixed(Judgment{Subject: "todo", ID: t.UUID, Verdict: verdictEscalate, DecidedBy: decidedByError, Reason: "no assignee tag"})
				continue
			}
			if j, ok := ruleJudgment(t, today); ok && t.Assignee != "" {
				fixed(j)
				continue
			}
			if slices.ContainsFunc(t.Links, func(l Link) bool { return l.Unreadable }) {
				fixed(Judgment{Subject: "todo", ID: t.UUID, Gate: gateTodoState.Name, Verdict: verdictEscalate,
					DecidedBy: decidedByError, Reason: "a source this todo cites is unreadable"})
				continue
			}
			def := gateTodoState
			if t.Assignee == "" {
				def = gateTodoOwner
			}
			if !cfg.Gate.TodoContextEnabled {
				fixed(Judgment{Subject: "todo", ID: t.UUID, Gate: def.Name, Verdict: verdictEscalate, DecidedBy: decidedByError, Reason: "todo context gates disabled pending evaluation"})
				continue
			}
			state, err := todoContext(cfg, p, t, tabs, today)
			if err != nil {
				fixed(Judgment{Subject: "todo", ID: t.UUID, Gate: def.Name, Verdict: verdictEscalate, DecidedBy: decidedByError, Reason: err.Error()})
				continue
			}
			jobs = append(jobs, func() Judgment { return gate.Judge(ctx, "todo", t.UUID, def, state) })
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
