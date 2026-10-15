package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"slices"
	"time"
)

type evaluationCase struct {
	UUID string `json:"uuid"`
	Gate string `json:"gate"`
}
type evaluationResult struct {
	UUID     string    `json:"uuid"`
	State    todoState `json:"state"`
	Judgment Judgment  `json:"judgment"`
}

func evaluateTodos(ctx context.Context, cfg Config, gate *Gate, now time.Time, path string) ([]evaluationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cases []evaluationCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, err
	}
	r := runner{cmds: cfg.Commands}
	todos, err := r.todos(ctx, now, 100*365*24*time.Hour)
	if err != nil {
		return nil, err
	}
	snapshot := collect(ctx, cfg, nil, now)
	out := make([]evaluationResult, len(cases))
	errs := make([]error, len(cases))
	parallel(cases, func(i int, item evaluationCase) {
		var todo Todo
		for _, candidate := range todos {
			if candidate.UUID == item.UUID {
				todo = candidate
				break
			}
		}
		if todo.UUID == "" {
			errs[i] = fmt.Errorf("evaluation todo %s absent from export", item.UUID)
			return
		}
		def := gateTodoState
		switch item.Gate {
		case gateTodoState.Name:
		case gateTodoOwner.Name:
			def = gateTodoOwner
		default:
			errs[i] = fmt.Errorf("unknown evaluation gate %q", item.Gate)
			return
		}
		state, err := evaluationState(ctx, cfg, snapshot.Projects, todo, now.Format(time.DateOnly))
		out[i] = evaluationResult{UUID: item.UUID, State: state}
		if err != nil {
			out[i].Judgment = Judgment{Subject: "todo", ID: item.UUID, Gate: def.Name, Verdict: verdictEscalate, DecidedBy: decidedByError, Reason: err.Error()}
			return
		}
		for _, link := range state.Links {
			if link.Unreadable {
				err = fmt.Errorf("a source this todo cites is unreadable")
			}
		}
		if err != nil {
			out[i].Judgment = Judgment{Subject: "todo", ID: item.UUID, Gate: def.Name, Verdict: verdictEscalate, DecidedBy: decidedByError, Reason: err.Error()}
			return
		}
		out[i].Judgment = gate.Judge(ctx, "todo", item.UUID, def, state)
	})
	return out, errors.Join(errs...)
}

func evaluationState(ctx context.Context, cfg Config, projects []Project, todo Todo, today string) (todoState, error) {
	todo.Links = slices.Clone(todo.Links)
	p := &Project{ID: todo.slug(), Label: todo.Domain}
	for i := range projects {
		if projects[i].ID == todo.slug() || (todo.slug() == "" && projects[i].ID == noProject) {
			p = &projects[i]
			break
		}
	}
	r := runner{cmds: cfg.Commands}
	for i := range todo.Links {
		link := &todo.Links[i]
		switch link.Kind {
		case "issue", "pr":
			if err := r.fillLink(ctx, link); err != nil {
				link.Unreadable = true
			}
		case "commit":
			if p.Root != "" {
				link.State = r.onDefault(ctx, p.Root, p.DefaultBranch, link.SHA)
			}
		}
	}
	var tabs []tabBrief
	for _, tab := range p.Tabs {
		tabs = append(tabs, tabBrief{Label: tab.Label, Agent: tab.Agent, State: tab.State, AgeS: tab.AgeS})
	}
	return todoContext(cfg, p, todo, tabs, today)
}
