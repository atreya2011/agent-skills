# Import

Do this once, when the local file has an Import section. The section names the previous tracker's agent. The CTO talks to the tracker by chat only and never edits its files.

Steps 1 to 4 take the lists. After step 4, write `Import lists taken: <date>` into the Import section. When that line already exists, skip steps 1 to 4 and run only steps 5 and 6.

1. Send the tracker one message through `herdr agent prompt <target> --wait --timeout 120000`. Ask for three lists: every open todo it tracks with its sources, every standing ruling word for word, and every pending wiki-edit proposal with its page and claim.
2. Read the reply with `herdr agent read <target> --source recent-unwrapped --lines 200`.
3. Compare the todo list with `task export`. Add each missing todo with `task add`. Annotate a todo that exists but lacks a source the tracker cites.
4. Append each ruling word for word to the Rulings section of the local file, dated. Record each wiki-edit proposal there as a `Pending wiki proposal` line. Do not apply a proposal. Before any edit lands, the Wiki writes rule re-verifies it against today's vault.
5. Ask the tracker the gate's own question, with the same `--wait --timeout 120000`: "Are your tabs still needed? Reply done only if you have listed everything and your tab can close." Pipe its reply into `bin/cto-snapshot reply`. If the verdict is `act` and the label is `done`, close its tab with `herdr tab close <tab_id>`. A timeout, a `blocked` state or any other verdict is "not done": report it in Needs you and keep the Import section for the next sweep.
6. After the tab is closed, delete the Import section from the local file.

The imported rulings can include the `Assignee split` ruling, so the sweep applies the split after this import.
