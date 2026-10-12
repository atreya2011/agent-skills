# Resume a dead session

Use this file when a session's live tab is absent.

## Placement

Decide by the session's cwd:

- If a workspace was created at that cwd (its `identity_cwd`) → open a **new tab** in
  that workspace (keeps same-project work together).
- Otherwise → open a **new workspace** (its own tab).

Match only on each workspace's `identity_cwd` (its creation dir, from
`~/.config/herdr/session.json`) — a tab's live cwd can drift after a `cd` into a
subdir. Normalize both sides (`realpath`, strip trailing slash) and, when
several workspaces share the cwd, pick the lowest-numbered **live** one so the choice
is deterministic. `agent start` requires a pane ID, so take the new tab's root
pane ID from the create response, then start the agent:

```bash
REPO=$(python3 -c 'import os,sys;print(os.path.realpath(sys.argv[1]).rstrip("/"))' "<path>")
# lowest-numbered live workspace whose identity_cwd == REPO; empty => none
WS=$(python3 - "$REPO" <<'PY'
import json,os,sys,subprocess
repo=sys.argv[1]
sess=json.load(open(os.path.expanduser("~/.config/herdr/session.json")))["workspaces"]
live={w["workspace_id"]:w["number"] for w in
      json.loads(subprocess.check_output(["herdr","workspace","list"]))["result"]["workspaces"]}
def norm(p): return os.path.realpath(p).rstrip("/") if p else ""
hits=sorted((w["id"] for w in sess if w["id"] in live and norm(w.get("identity_cwd"))==repo),
            key=lambda i: live[i])
print(hits[0] if hits else "")
PY
)

if [ -n "$WS" ]; then
    P=$(herdr tab create --workspace "$WS" --cwd "$REPO" --no-focus \
        | python3 -c 'import sys,json;print(json.load(sys.stdin)["result"]["root_pane"]["pane_id"])')
else
    P=$(herdr workspace create --cwd "$REPO" --no-focus \
        | python3 -c 'import sys,json;print(json.load(sys.stdin)["result"]["root_pane"]["pane_id"])')
fi
```

## Start the agent

Start the agent with a validated `agent start` — it targets the new tab's root pane
by ID, verifies the detected agent kind, and returns only once the agent is ready for
input (30 s default timeout). Names match `[a-z][a-z0-9_-]{0,31}`, stay unique
among live agents, and clear when the occupant exits. Native agent args go after `--`.

- **claude** — history in `~/.claude/projects/<slug>/*.jsonl`, where `<slug>` is the
  cwd with every non-alphanumeric char replaced by `-`.

  ```bash
  herdr agent start <name> --kind claude --pane "$P" -- --continue   # most recent in that cwd
  herdr agent start <name> --kind claude --pane "$P" -- --resume     # picker
  # list candidates: ls -t ~/.claude/projects/<slug>/*.jsonl
  ```

- **codex** — index in `~/.codex/session_index.jsonl` (`id`, `thread_name`,
  `updated_at`). A session's cwd lives inside its log at
  `~/.codex/sessions/YYYY/MM/DD/*.jsonl`.

  ```bash
  herdr agent start <name> --kind codex --pane "$P" -- resume --last   # most recent
  herdr agent start <name> --kind codex --pane "$P" -- resume          # picker
  herdr agent start <name> --kind codex --pane "$P" -- resume <uuid>   # exact session
  ```

Then hand it work: `herdr agent prompt <name> "<task>" --wait`.
