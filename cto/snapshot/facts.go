package main

import "time"

type Unreadable struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
	Error  string `json:"error"`
}

type Workspace struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	State           string `json:"state"`
	Pinned          bool   `json:"pinned"`
	OrchestratorTab string `json:"orchestrator_tab,omitempty"`
}

type Tab struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Workspace string `json:"workspace"`
	Agent     string `json:"agent,omitempty"`
	Target    string `json:"target,omitempty"`
	State     string `json:"state"`
	AgeS      *int   `json:"age_s"`
	Pinned    bool   `json:"pinned"`
	CWD       string `json:"cwd,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type PR struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	State    string    `json:"state"`
	Branch   string    `json:"branch"`
	Draft    bool      `json:"draft"`
	MergedAt time.Time `json:"merged_at,omitzero"`
	URL      string    `json:"url"`
}

type Link struct {
	Kind        string   `json:"kind"`
	URL         string   `json:"url,omitempty"`
	SHA         string   `json:"sha,omitempty"`
	State       string   `json:"state,omitempty"`
	StateReason string   `json:"state_reason,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Unreadable  bool     `json:"unreadable,omitzero"`

	slug string
}

type Todo struct {
	UUID        string    `json:"uuid"`
	Description string    `json:"description"`
	Domain      string    `json:"domain,omitempty"`
	Assignee    string    `json:"assignee,omitempty"`
	Ask         bool      `json:"ask"`
	Status      string    `json:"status"`
	Annotations []string  `json:"annotations"`
	Due         time.Time `json:"due,omitzero"`
	ClosedAt    string    `json:"closed_at,omitempty"`
	Links       []Link    `json:"links"`
	SessionID   string    `json:"session_id,omitempty"`
	Ready       bool      `json:"ready"`
	Dispatched  bool      `json:"dispatched"`
}

type Session struct {
	Profile  string    `json:"profile"`
	ID       string    `json:"id"`
	Modified time.Time `json:"modified"`
	AgeS     int       `json:"age_s"`
	CWD      string    `json:"cwd,omitempty"`
	TabID    string    `json:"tab_id,omitempty"`
	Excerpt  string    `json:"excerpt,omitempty"`
}

type ProfileInfo struct {
	Name             string    `json:"name"`
	Kind             string    `json:"kind"`
	LastActivity     time.Time `json:"last_activity,omitzero"`
	SessionsInWindow int       `json:"sessions_in_window"`
	SameAs           string    `json:"same_as,omitempty"`
}

type Vault struct {
	Path     string `json:"path"`
	Readable bool   `json:"readable"`
}
