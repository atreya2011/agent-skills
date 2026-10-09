package main

import "time"

// Unreadable names a source the snapshot could not read. A missing source is
// listed here and the run goes on, so silence in the snapshot never means that
// nothing happened.
type Unreadable struct {
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
	Error  string `json:"error"`
}

// Workspace is one herdr workspace.
type Workspace struct {
	ID              string `json:"id"`
	Label           string `json:"label"`
	State           string `json:"state"`
	Pinned          bool   `json:"pinned"`
	OrchestratorTab string `json:"orchestrator_tab,omitempty"`
}

// Tab is one herdr tab. Target is what herdr agent commands take: the agent's
// name, else its pane ID. AgeS is the seconds since the tab's transcript was
// last written, or null when the tab has no readable transcript, because herdr
// itself reports no state timestamp.
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

// PR is one GitHub pull request of a project. MergedAt is read into a time, so
// the output does not depend on how gh formats it.
type PR struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	State    string    `json:"state"`
	Branch   string    `json:"branch"`
	Draft    bool      `json:"draft"`
	MergedAt time.Time `json:"merged_at,omitzero"`
	URL      string    `json:"url"`
}

// Link is a fact source a todo cites: a GitHub issue or pull request, or a
// commit. Issue and PR links carry the state read from GitHub; a commit link
// says whether the commit is on the default branch.
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

// Todo is one Taskwarrior item. Domain is Taskwarrior's project field.
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

// Session is one agent session whose transcript was written within the window.
type Session struct {
	Profile  string    `json:"profile"`
	ID       string    `json:"id"`
	Modified time.Time `json:"modified"`
	AgeS     int       `json:"age_s"`
	CWD      string    `json:"cwd,omitempty"`
	TabID    string    `json:"tab_id,omitempty"`
	Excerpt  string    `json:"excerpt,omitempty"`
}

// ProfileInfo summarizes one profile's transcripts. SameAs names the profile
// whose directory this one shares through a symlink; the shared sessions are
// counted once, under that profile.
type ProfileInfo struct {
	Name             string    `json:"name"`
	Kind             string    `json:"kind"`
	LastActivity     time.Time `json:"last_activity,omitzero"`
	SessionsInWindow int       `json:"sessions_in_window"`
	SameAs           string    `json:"same_as,omitempty"`
}

// Vault describes the wiki vault mapped to a project.
type Vault struct {
	Path     string `json:"path"`
	Readable bool   `json:"readable"`
}
