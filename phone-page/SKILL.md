---
name: phone-page
description: Publish designed read pages and tappable question pages for replies that do not fit cleanly in chat on a phone.
---

# Phone page

Skip the page when you can explain the reply in 50 words or less.

Use a page when a reply to the user needs more than two sentences, a diagram, a table or any question. Put every question to the user on a question page, never in reply text or the agent CLI's question dialog. Chat between agents is unaffected.

## Local file

`~/.agents/local/phone-page.md` holds the store command and nothing else. Read it when it exists. The store keeps pages and their latest submissions; the cf-page-store skill provides one. Read [references/install.md](references/install.md) before installing this skill or writing the local file.

## Write a page

- Before writing any page, use `git ls-files` in the current repository to find files named `glossary.md` at any path, ignoring letter case, and read any matches.
- Use glossary terms with their exact spelling and meaning, and define each term on first use using the glossary's own definition.
- A page covers one project; questions or content about different projects go on separate pages; the page title names the project.
- Follow Grice's [maxim of quantity](https://en.wikipedia.org/wiki/Cooperative_principle#Maxim_of_quantity).
- For a read page, read [references/design.md](references/design.md) before writing it, then publish it with `phone-page publish <file>`.
- For a question page, read [references/question-spec.md](references/question-spec.md) before writing the spec, then publish it with `phone-page ask <spec>`; do not hand-write its form.
- Give each question exactly what the user needs to decide, with no wall of text and no flow-only page:
  - the problem, what was tried, the result, and what needs deciding, then the recommendation with its one-line reason, as the first context bullets;
  - the facts it rests on in a `table`, with numbers and sources;
  - known limits as two or three one-line `cards`, or as a table row when there is only one;
  - the process as a `flow` or `flows` chart with short boxes;
  - each option's consequence in its description.
- Add nothing beyond those parts, and never cut a fact the decision depends on.

Read [references/commands.md](references/commands.md) before running any other `phone-page` command, and when a command fails.

## Reply and continue

- After publishing, reply in one line that says what the page contains and what the user should do, followed by its link. A chat reply that points to a page has at most two sentences plus the link, with no code block, table or list.
- After the user replies `done`, or when the store's listener prompts this tab about a new submission, run `phone-page answers <id>` and continue from that submission.
- When the user asks for a change to a page they already have, rebuild it in place with `phone-page ask <spec> <id>` or `phone-page publish <file> <id>` and reply with the same link. Never publish a replacement page for content the user already has a link to. Replacing a page discards its previous submission.
