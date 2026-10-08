package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	githubLink = regexp.MustCompile(`https://github\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/(issues|pull)/([0-9]+)`)
	commitRef  = regexp.MustCompile(`\bcommit:\s*([0-9a-f]{7,40})\b`)
	sessionRef = regexp.MustCompile(`session:\s*\S*?([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
)

// taskTimeLayout is Taskwarrior's export timestamp format.
const taskTimeLayout = "20060102T150405Z"

type taskItem struct {
	UUID        string   `json:"uuid"`
	Description string   `json:"description"`
	Project     string   `json:"project"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
	Due         string   `json:"due"`
	End         string   `json:"end"`
	Annotations []struct {
		Description string `json:"description"`
	} `json:"annotations"`
}

// todos runs "<task> export" and returns the open todos (pending or waiting)
// and the todos completed within the window.
func (r runner) todos(ctx context.Context, now time.Time, window time.Duration) ([]Todo, error) {
	out, _, err := r.output(ctx, "", r.cmds.Task, "export")
	if err != nil {
		return nil, err
	}
	var items []taskItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("task export: unreadable output: %w", err)
	}
	var todos []Todo
	for _, it := range items {
		closedAt := ""
		switch it.Status {
		case "pending", "waiting":
		case "completed":
			end, err := time.Parse(taskTimeLayout, it.End)
			if err != nil || now.Sub(end) > window {
				continue
			}
			closedAt = end.Format(time.RFC3339)
		default:
			continue
		}
		todos = append(todos, parseTodo(it, closedAt))
	}
	return todos, nil
}

// ownAnnotation reports whether the CTO wrote an annotation itself: a pending
// question ("ask:") or a recorded verdict ("cto "). Such an annotation can name
// a pull request, a commit or a session without citing it as a source, so the
// link scans and the gate input skip it.
func ownAnnotation(a string) bool {
	return strings.HasPrefix(a, "ask:") || strings.HasPrefix(a, "cto ")
}

// parseTodo maps one export item to a Todo and extracts the links its text and
// annotations cite.
func parseTodo(it taskItem, closedAt string) Todo {
	t := Todo{
		UUID: it.UUID, Description: it.Description, Domain: it.Project, Status: it.Status,
		Due: it.Due, ClosedAt: closedAt, Annotations: []string{}, Links: []Link{},
	}
	for _, tag := range it.Tags {
		switch tag {
		case "ask":
			t.Ask = true
		case "cos", "cto":
			if t.Assignee == "" {
				t.Assignee = tag
			}
		}
	}
	texts := []string{it.Description}
	for _, a := range it.Annotations {
		t.Annotations = append(t.Annotations, a.Description)
		if ownAnnotation(a.Description) {
			continue
		}
		texts = append(texts, a.Description)
		if strings.HasPrefix(a.Description, "run:") {
			t.Dispatched = true
		}
	}
	seen := map[string]bool{}
	for _, text := range texts {
		for _, m := range githubLink.FindAllStringSubmatch(text, -1) {
			if seen[m[0]] {
				continue
			}
			seen[m[0]] = true
			kind := "issue"
			if m[2] == "pull" {
				kind = "pr"
			}
			t.Links = append(t.Links, Link{Kind: kind, URL: m[0], slug: m[1]})
		}
		for _, m := range commitRef.FindAllStringSubmatch(text, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				t.Links = append(t.Links, Link{Kind: "commit", SHA: m[1], State: "unchecked"})
			}
		}
		if m := sessionRef.FindStringSubmatch(text); m != nil && t.SessionID == "" {
			t.SessionID = m[1]
		}
	}
	return t
}

// slug returns the owner/name of the first GitHub link, which places the todo
// in a project.
func (t Todo) slug() string {
	for _, l := range t.Links {
		if l.slug != "" {
			return l.slug
		}
	}
	return ""
}
