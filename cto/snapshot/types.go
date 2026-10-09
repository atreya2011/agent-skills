package main

import "time"

// noProject is the ID of the group that holds tabs, sessions and todos that
// belong to no repository or directory.
const noProject = "none"

// Snapshot is the JSON document the tool prints: the facts of one sweep, the
// sources it could not read, and the judgments of the rules and gates.
type Snapshot struct {
	GeneratedAt time.Time     `json:"generated_at"`
	WindowHours int           `json:"window_hours"`
	Workspaces  []Workspace   `json:"workspaces"`
	Projects    []Project     `json:"projects"`
	Profiles    []ProfileInfo `json:"profiles"`
	Unreadable  []Unreadable  `json:"unreadable"`
	Judgments   []Judgment    `json:"judgments"`
}

// Project is a repository or working directory that sessions work in. Its ID
// is the GitHub owner/name of its origin, else its root path or directory. A
// project without Root has no git facts.
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
