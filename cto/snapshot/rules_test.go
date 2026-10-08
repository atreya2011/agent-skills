package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJudged(t *testing.T) {
	tests := []struct {
		name string
		todo Todo
		want bool
	}{
		{"a pending cto todo", Todo{Assignee: "cto", Status: "pending"}, true},
		{"an untagged todo, before the first-sweep split", Todo{Status: "pending"}, true},
		{"a waiting cto todo", Todo{Assignee: "cto", Status: "waiting"}, true},
		{"a cos todo is never touched", Todo{Assignee: "cos", Status: "pending"}, false},
		{"a completed todo", Todo{Assignee: "cto", Status: "completed"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, judged(tt.todo))
		})
	}
}

func TestRuleJudgment(t *testing.T) {
	tests := []struct {
		name     string
		links    []Link
		wantRule string
	}{
		{"no links", nil, ""},
		{"a merged pull request", []Link{{Kind: "pr", State: "MERGED"}}, "pr_merged"},
		{"a pull request closed without merging", []Link{{Kind: "pr", State: "CLOSED"}}, ""},
		{"an open pull request", []Link{{Kind: "pr", State: "OPEN"}}, ""},
		{"an issue closed as completed", []Link{{Kind: "issue", State: "CLOSED", StateReason: "COMPLETED"}}, "issue_closed"},
		{"an issue closed with no reason recorded", []Link{{Kind: "issue", State: "CLOSED"}}, "issue_closed"},
		{"an issue closed as not planned is not proof of the work", []Link{{Kind: "issue", State: "CLOSED", StateReason: "NOT_PLANNED"}}, ""},
		{"an open issue", []Link{{Kind: "issue", State: "OPEN"}}, ""},
		{"a commit on the default branch", []Link{{Kind: "commit", State: "on_default"}}, "commit_on_default"},
		{"a commit off the default branch", []Link{{Kind: "commit", State: "not_on_default"}}, ""},
		{"a commit that could not be checked", []Link{{Kind: "commit", State: "unchecked"}}, ""},
		{"proof beside an unreadable link still counts", []Link{{Kind: "issue", Unreadable: true}, {Kind: "pr", State: "MERGED"}}, "pr_merged"},
		{"the first rule that fires wins", []Link{{Kind: "commit", State: "on_default"}, {Kind: "pr", State: "MERGED"}}, "commit_on_default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j, ok := ruleJudgment(Todo{UUID: "u1", Links: tt.links}, "2026-10-08")
			assert.Equal(t, tt.wantRule != "", ok)
			if !ok {
				return
			}
			assert.Equal(t, Judgment{
				Subject: "todo", ID: "u1", Label: "done", Confidence: 1, Verdict: verdictAct, DecidedBy: decidedByRule,
				Rule: tt.wantRule, Reason: j.Reason, Annotation: "cto rule " + tt.wantRule + " done 1.00 act 2026-10-08",
			}, j)
			assert.NotEmpty(t, j.Reason)
		})
	}
}
