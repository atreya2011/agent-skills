---
name: find-unknowns
description: Map unknown unknowns before work by exploring the codebase and domain in parallel, teaching the findings, and writing a persistent map under docs/unknowns. Use for a blindspot pass, when the user asks what they are missing, or admits unfamiliarity before starting.
---

# find-unknowns

A blindspot pass, after Thariq's "Field Guide to Fable": the map (the user's prompt and context) always lags the territory (the codebase, the domain, its real constraints). The gap is unknowns. An interview or plan review can only probe questions somebody already thought to raise — this skill finds the questions nobody raised, teaches them to the user, and records them so a later planning or /grill-me session starts from a wider map.

This skill ends at discovery and teaching. Stop there: hand the findings to the user and let them decide on implementation, a plan, or invoking other skills next.

## Workflow

### 1. Establish the starting point

The pass is calibrated by two facts only the user knows: what the task is, and how much they already know about the territory (both the domain and this codebase's corner of it).

- If the invocation already states both ("I'm adding an auth provider but know nothing about the auth modules"), skip straight to exploration, using exactly what was already said.
- If either is missing, ask once via `AskUserQuestion` — a single batched call, at most 3 questions (typically: what's the task, experience with the domain, familiarity with this code area). Treat this as the only intake round, then explore.

### 2. Explore the territory in parallel

Fan out read-only `Explore` subagents, one per independent angle, in a single message so they run concurrently. Pick angles that apply to the task:

- **Target area** — the code the task will touch: entry points, data flow, existing conventions, hidden invariants.
- **History** — git log/blame of the target area: past attempts, reverts, TODO/FIXME graveyards, why things are the way they are.
- **Adjacent constraints** — modules that call into or depend on the target area; configs, migrations, CI, feature flags that would silently break.
- **Domain research** — when the unfamiliarity is about the domain itself (color grading, OAuth flows, a protocol), use web search and docs rather than the repo.

Skip inapplicable angles (a pure-domain task has no git history to read), but say which were skipped and why. Each subagent's brief: "surface what someone new to this area wouldn't know to ask about — pitfalls, invariants, prior art, constraints — not a general summary."

### 3. Synthesize and teach

From the subagent reports, extract the genuine unknown unknowns: things the user, given their stated starting point, would not have known to ask about. Filter aggressively — a finding the user obviously already knows is noise, and ten vague findings teach less than five sharp ones.

Teach the findings in chat, in plain prose (complete sentences — this is explanation, not a status report). For each finding:

- Name it.
- Explain what it is and how the territory actually behaves.
- Say why it bites: the concrete way it would derail the task if discovered mid-implementation.
- State the question it raises — the decision the user now needs to make.

Order by danger: the finding most likely to invalidate the user's current approach comes first. If a finding is fatal to the stated approach, say so bluntly before anything else.

### 4. Write the map

After teaching, write the map file:

- **Location:** `docs/unknowns/<topic>.md` in the repo root (create the directory if needed). Outside a git repo, write `unknowns-<topic>.md` in the cwd. `<topic>` is a short kebab-case slug of the task.
- **Preserve every existing map.** Check whether the path exists first; on collision, suffix `-2`, `-3`, … A previous map is a record of a previous state of knowledge — it must survive.

Use exactly this structure:

```markdown
# Unknowns map: <topic>

Date: <date> · Starting point: <one line: task + user's stated experience>

## Territory explored
<bullets: which angles ran, what each covered, which were skipped and why>

## Unknown unknowns
### <finding name>
<what it is, why it bites, evidence (file:line, doc link)>
<repeat per finding, danger-ordered>

## Open questions for grilling
<numbered list: one decision-shaped question per finding that needs a user
choice — phrased so an interviewer could ask it verbatim>
```

### 5. Close

End by stating the map path and that the "Open questions for grilling" section is ready to feed a planning or /grill-me session. Leave the choice of what happens next to the user.
