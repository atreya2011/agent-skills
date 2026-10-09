package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"time"
)

type ghPR struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	State       string    `json:"state"`
	HeadRefName string    `json:"headRefName"`
	IsDraft     bool      `json:"isDraft"`
	MergedAt    time.Time `json:"mergedAt"`
	URL         string    `json:"url"`
}

type ghLink struct {
	State       string `json:"state"`
	StateReason string `json:"stateReason"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// pullRequests returns the project's 30 most recently updated pull requests,
// open or closed, from "gh pr list".
func (r runner) pullRequests(ctx context.Context, slug string) ([]PR, error) {
	out, _, err := r.output(ctx, "", r.cmds.Gh, "pr", "list", "--repo", slug, "--state", "all", "--limit", "30",
		"--json", "number,title,state,headRefName,isDraft,mergedAt,url")
	if err != nil {
		return nil, err
	}
	var raw []ghPR
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("gh pr list: unreadable output: %w", err)
	}
	prs := []PR{}
	for _, p := range raw {
		prs = append(prs, PR{Number: p.Number, Title: p.Title, State: p.State, Branch: p.HeadRefName,
			Draft: p.IsDraft, MergedAt: p.MergedAt.UTC(), URL: p.URL})
	}
	return prs, nil
}

// fillLink reads the state of a linked issue or pull request from GitHub.
func (r runner) fillLink(ctx context.Context, l *Link) error {
	noun, fields := "issue", "state,stateReason,labels"
	if l.Kind == "pr" {
		noun, fields = "pr", "state"
	}
	out, _, err := r.output(ctx, "", r.cmds.Gh, noun, "view", l.URL, "--json", fields)
	if err != nil {
		return err
	}
	var raw ghLink
	if err := json.Unmarshal(out, &raw); err != nil {
		return fmt.Errorf("gh %s view: unreadable output: %w", noun, err)
	}
	l.State, l.StateReason = raw.State, raw.StateReason
	for _, label := range raw.Labels {
		l.Labels = append(l.Labels, label.Name)
	}
	return nil
}
