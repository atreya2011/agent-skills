package main

import (
	"fmt"
	"slices"
)

func open(t Todo) bool {
	return t.Status == "pending" || t.Status == "waiting"
}

func judged(t Todo) bool {
	return open(t) && t.Assignee != "cos"
}

func ruleJudgment(t Todo, today string) (Judgment, bool) {
	if slices.ContainsFunc(t.Links, func(l Link) bool {
		return (l.Kind == "issue" || l.Kind == "pr") && (l.Unreadable || l.State == "OPEN")
	}) {
		return Judgment{}, false
	}
	for _, l := range t.Links {
		var rule, why string
		switch {
		case l.Kind == "pr" && l.State == "MERGED":
			rule, why = "pr_merged", "pull request "+l.URL+" is merged"
		case l.Kind == "issue" && l.State == "CLOSED" && (l.StateReason == "COMPLETED" || l.StateReason == ""):
			rule, why = "issue_closed", "issue "+l.URL+" is closed as completed"
		case l.Kind == "commit" && l.State == "on_default":
			rule, why = "commit_on_default", "commit "+l.SHA+" is on the default branch"
		default:
			continue
		}
		return Judgment{
			Subject: "todo", ID: t.UUID, Label: "done", Confidence: 1, Verdict: verdictAct,
			DecidedBy: decidedByRule, Rule: rule, Reason: why,
			Annotation: fmt.Sprintf("cto rule %s done 1.00 act %s", rule, today),
		}, true
	}
	return Judgment{}, false
}
