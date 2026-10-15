---
name: find-unknowns
description: Map unknown unknowns before work by exploring the codebase and domain in parallel, teaching the findings, and writing a persistent map under docs/unknowns. Use for a blindspot pass, when the user asks what they are missing, or admits unfamiliarity before starting.
---

# find-unknowns

After Thariq's "Field Guide to Fable": prompts lag the codebase, domain and constraints. Interviews and plan reviews probe already-raised questions. Find unraised questions, teach them, and record them for later planning or /grill-me sessions.

Stop after discovery and teaching. Let the user choose implementation, planning, or another skill.

## Workflow

### 1. Establish the starting point

Two facts calibrate the pass: what the task is, and how much the user knows about the domain and this codebase's part of it.

- If the invocation states both, start exploring with exactly what was said.
- If either is missing, use one phone-page intake round: ask one question at a time about the task, domain experience or code familiarity; batch at most 3 only when the user requests all remaining questions or runs cto. Then explore.

### 2. Explore in parallel

Spawn read-only `Explore` subagents in a single message, one per angle that applies:

- **Target area**: the code the task will touch: entry points, data flow, conventions, hidden invariants.
- **History**: git log and blame of the target area: past attempts, reverts, TODO and FIXME comments, reasons for the current design.
- **Adjacent constraints**: modules that call or depend on the target area, and configs, migrations, CI and feature flags that would break silently.
- **Domain research**: web search and docs instead of the repo, when the unfamiliarity is about the domain itself.

Skip angles that do not apply, and say which and why. Brief each subagent: "Surface what someone new to this area would not know to ask about: pitfalls, invariants, prior art, constraints. Do not write a general summary."

### 3. Synthesize and teach

Keep only real unknown unknowns: things the user, from their stated starting point, would not have known to ask about. Drop findings the user obviously knows; five sharp findings teach more than ten vague ones.

Explain findings in complete sentences. Follow [phone-page](../phone-page/SKILL.md) only when a finding needs the user's decision or the user explicitly asked for the explanation, and never in a headless run such as `codex exec` or `claude -p` or in a run another agent dispatched, including a subagent. For each finding:

- Name it.
- Explain what it is and how the territory actually behaves.
- Say how it would derail the task if found mid-implementation.
- State the decision it raises for the user.

Put the finding most likely to invalidate the user's approach first. If a finding is fatal to the stated approach, say so bluntly before anything else.

### 4. Write the map

After teaching, read [references/map.md](references/map.md) before writing the map; follow its path, collision rule and structure. Never overwrite an existing map.

### 5. Close

State the map path and say that its "Open questions for grilling" section is ready for a planning or /grill-me session. Leave the next step to the user.
