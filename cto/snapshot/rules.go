package main

import "fmt"

// judged reports whether the CTO judges a todo: it is open and is tagged for
// the CTO or not tagged yet. A +cos todo belongs to the CoS and is never
// touched.
func judged(t Todo) bool {
	return (t.Assignee == "cto" || t.Assignee == "") && (t.Status == "pending" || t.Status == "waiting")
}

// ruleJudgment applies proof as CONTEXT.md defines it, with no model call: a
// linked pull request is merged, a linked issue is closed as completed, or a
// cited commit is on the default branch. It returns the first rule that fires.
func ruleJudgment(t Todo, today string) (Judgment, bool) {
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
