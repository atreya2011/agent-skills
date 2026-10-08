package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"
)

type herdrWorkspace struct {
	ID          string `json:"workspace_id"`
	Label       string `json:"label"`
	AgentStatus string `json:"agent_status"`
}

type herdrTab struct {
	ID          string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
	AgentStatus string `json:"agent_status"`
}

type herdrAgent struct {
	Agent        string `json:"agent"`
	Name         string `json:"name"`
	PaneID       string `json:"pane_id"`
	TabID        string `json:"tab_id"`
	CWD          string `json:"cwd"`
	AgentSession struct {
		Value string `json:"value"`
	} `json:"agent_session"`
}

// herdrLists is the part of a "herdr <noun> list" reply the snapshot reads.
type herdrLists struct {
	Result struct {
		Workspaces []herdrWorkspace `json:"workspaces"`
		Tabs       []herdrTab       `json:"tabs"`
		Agents     []herdrAgent     `json:"agents"`
	} `json:"result"`
}

// herdrList runs "herdr <noun> list" and decodes its JSON reply.
func (r runner) herdrList(ctx context.Context, noun string) (herdrLists, error) {
	out, _, err := r.output(ctx, "", r.cmds.Herdr, noun, "list")
	if err != nil {
		return herdrLists{}, err
	}
	var lists herdrLists
	if err := json.Unmarshal(out, &lists); err != nil {
		return herdrLists{}, fmt.Errorf("herdr %s list: unreadable output: %w", noun, err)
	}
	return lists, nil
}

// buildHerdr joins the three herdr lists. Each tab takes the first agent in it
// by pane ID; a tab without an agent is a plain shell. A workspace's
// orchestrator is the tab labeled "orchestrator".
func buildHerdr(cfg Config, workspaces []herdrWorkspace, tabs []herdrTab, agents []herdrAgent) ([]Workspace, []Tab) {
	label := map[string]string{}
	var outWS []Workspace
	for _, w := range workspaces {
		label[w.ID] = w.Label
		outWS = append(outWS, Workspace{
			ID: w.ID, Label: w.Label, State: w.AgentStatus,
			Pinned: slices.Contains(cfg.Pinned.Workspaces, w.Label),
		})
	}
	agentOf := map[string]herdrAgent{}
	for _, a := range agents {
		if first, ok := agentOf[a.TabID]; !ok || a.PaneID < first.PaneID {
			agentOf[a.TabID] = a
		}
	}
	var outTabs []Tab
	for _, t := range tabs {
		a := agentOf[t.ID]
		target := a.Name
		if target == "" {
			target = a.PaneID
		}
		outTabs = append(outTabs, Tab{
			ID: t.ID, Label: t.Label, Workspace: t.WorkspaceID,
			Agent: a.Agent, Target: target, State: t.AgentStatus,
			Pinned: slices.Contains(cfg.Pinned.Tabs, t.Label) || slices.Contains(cfg.Pinned.Workspaces, label[t.WorkspaceID]),
			CWD:    a.CWD, SessionID: a.AgentSession.Value,
		})
		if t.Label == "orchestrator" {
			for i := range outWS {
				if outWS[i].ID == t.WorkspaceID {
					outWS[i].OrchestratorTab = t.ID
				}
			}
		}
	}
	return outWS, outTabs
}
