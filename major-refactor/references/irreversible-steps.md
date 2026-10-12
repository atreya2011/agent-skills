# Irreversible steps

Use this protocol for every destructive or irreversible step: production data, schema changes, deletions at scale. The user approves a specific list, not an idea.

1. **Restore point first.** Create a git tag, and verify a snapshot or backup suited to the backend: a child audits the backup inventory for count, recency and retention.
2. **Dry-run inventory.** A child produces the exact list of what the step will touch: row counts, file lists, object IDs. Give numbers, not intentions.
3. **The user approves the inventory** on a phone-page question page.
4. **Execute exactly that inventory.** If reality diverges from the approved inventory mid-run, stop and escalate; do not decide alone.
5. **Re-verify counts** after execution and record them in the manifest.
