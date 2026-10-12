# Unknowns map

## Path

- In a git repo, write `docs/unknowns/<topic>.md` under the repo root, and create the directory if needed. Outside a git repo, write `unknowns-<topic>.md` in the current directory. `<topic>` is a short kebab-case slug of the task.
- Preserve every existing map. Check whether the path exists first; on a collision, add the suffix `-2`, `-3`, and so on. A previous map records a previous state of knowledge and must survive.

## Structure

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
