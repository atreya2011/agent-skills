---
name: phone-page
description: Publish designed read pages and tappable question pages in an interactive session, only for an item that needs the user's decision or an explanation the user explicitly asked for. Never use it for status summaries or job reports, in headless runs (codex exec, claude -p), or in agent-dispatched or subagent runs.
---

# Phone page

Use a page only in an interactive session, and only for an item that needs the user's decision or for an explanation the user explicitly asked for. Never use a page for a status summary or job report, in a headless run such as `codex exec` or `claude -p`, or in a run that another agent dispatched, including a subagent. Within that scope, put every question to the user on a question page, never in reply text or the agent CLI's question dialog. Chat between agents is unaffected.

## Local file

`~/.agents/local/phone-page.md` holds the store command and nothing else. Read it when it exists. The store keeps pages and their latest submissions; the cf-page-store skill provides one. Read [references/install.md](references/install.md) before installing this skill or writing the local file.

## Write a page

- Before writing any page, use `git ls-files` in the current repository to find files named `glossary.md` at any path, ignoring letter case, and read any matches.
- Use glossary terms with their exact spelling and meaning, and define each term on first use using the glossary's own definition.
- A page covers one project; questions or content about different projects go on separate pages; the page title names the project.
- Follow Grice's [maxim of quantity](https://en.wikipedia.org/wiki/Cooperative_principle#Maxim_of_quantity).
- Read [references/design.md](references/design.md) for presentation and [references/blocks.md](references/blocks.md) for shared content fields.
- Write read pages only as [read specs](references/read-spec.md), then run `phone-page read <spec>`; never hand-write their HTML.
- Write question pages as [question specs](references/question-spec.md), then run `phone-page ask <spec>`; never hand-write their forms.
- Give each question exactly what the user needs to decide, with no wall of text and no flow-only page:
  - the problem, what was tried, the result, and what needs deciding, then the recommendation with its one-line reason, as the first context bullets;
  - the facts it rests on in a `table`, with numbers and sources;
  - known limits as two or three one-line `cards`, or as a table row when there is only one;
  - the process as a `flow` or `flows` chart, rendered through Mermaid with short boxes and decision diamonds, or a `mermaid` block for diagrams a flow cannot express;
  - each option's consequence in its description.
- Add nothing beyond those parts, and never cut a fact the decision depends on.

Read [references/commands.md](references/commands.md) before running any other `phone-page` command, and when a command fails.

## Reply and continue

- After publishing, run `phone-page attach` so the page is annotated on the task of its asking tab; read [references/tab-tasks.md](references/tab-tasks.md) for the tab-task model, `herdr-tab-sync` and `phone-page close`.
- Then reply in one line that says what the page contains and what the user should do, followed by its link. A chat reply that points to a page has at most two sentences plus the link, with no code block, table or list.
- After the user replies `done`, or when the store's listener prompts this tab about a new submission, run `phone-page answers <id>` and continue from that submission.
- When the user asks for a change to a page they already have, rebuild it in place with `phone-page ask <spec> <id>` or `phone-page read <spec> <id>` and reply with the same link. Never publish a replacement page for content the user already has a link to. Replacing a page discards its previous submission.
