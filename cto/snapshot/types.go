package main

import "time"

const noProject = "none"

type Snapshot struct {
	GeneratedAt time.Time     `json:"generated_at"`
	WindowHours int           `json:"window_hours"`
	Workspaces  []Workspace   `json:"workspaces"`
	Projects    []Project     `json:"projects"`
	Profiles    []ProfileInfo `json:"profiles"`
	Unreadable  []Unreadable  `json:"unreadable"`
	Judgments   []Judgment    `json:"judgments"`
}

type Project struct {
	ID            string    `json:"id"`
	Label         string    `json:"label"`
	Root          string    `json:"root,omitempty"`
	Branch        string    `json:"branch,omitempty"`
	Dirty         int       `json:"dirty"`
	Ahead         int       `json:"ahead"`
	Behind        int       `json:"behind"`
	DefaultBranch string    `json:"default_branch,omitempty"`
	PRs           []PR      `json:"prs"`
	Tabs          []Tab     `json:"tabs"`
	Todos         []Todo    `json:"todos"`
	Sessions      []Session `json:"sessions"`
	Vault         *Vault    `json:"vault,omitempty"`

	slug string
}
