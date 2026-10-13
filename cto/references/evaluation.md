# Read-only evaluation


Create a JSON case list with one object per todo: `{"uuid":"<full todo UUID>","gate":"todo_state"}` or `{"uuid":"<full todo UUID>","gate":"todo_owner"}`. Run `bin/cto-snapshot evaluate --cases <case list> --local-file <evaluation local file>`. It reads Taskwarrior and current linked sources, prints the exact context and judgments, and makes no Taskwarrior writes. The command bypasses the rollout flag, existing owner tags and proof rules so each requested gate is measured directly. Keep expected answers outside the input.

Compare state answers with the known state and owner answers with the known owner. A substantive wrong answer at or above `thresholds.floor` fails the evaluation, even when the higher action threshold would escalate it. A confidently selected single owner for a task that needs splitting is wrong. Unknown, errors and answers below the floor go to the user. Report todo, expected answer, verdict, confidence and correct, wrong or sent to user; enable neither todo gate after a failure.
