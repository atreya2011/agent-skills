package main

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
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
	// A wide window retains completed todos so evaluation is independent of later live status changes.
	todos, err := r.todos(ctx, now, 100*365*24*time.Hour)
	if err != nil {
		return nil, err
	}
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
		for li := range todo.Links {
			link := &todo.Links[li]
			if link.Kind == "issue" || link.Kind == "pr" {
				if err := r.fillLink(ctx, link); err != nil {
					link.Unreadable = true
				}
			}
		}
		p := &Project{ID: todo.slug(), Label: todo.Domain}
		state, err := todoContext(cfg, p, todo, nil, now.Format(time.DateOnly))
		out[i] = evaluationResult{UUID: item.UUID, State: state}
		if err != nil {
			out[i].Judgment = Judgment{Subject: "todo", ID: item.UUID, Gate: def.Name, Verdict: verdictEscalate, DecidedBy: decidedByError, Reason: err.Error()}
			return
		}
		for _, link := range todo.Links {
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
