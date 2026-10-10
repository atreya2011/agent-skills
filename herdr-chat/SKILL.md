---
name: herdr-chat
description: Talk to another agent through its herdr tab, by chat only. Use when the user asks to discuss with, ask, brief or check on another tab or agent.
---

# herdr-chat

Chat is the only channel to another agent's tab.

## Rules

- Never edit the other tab's files. Never hand it a file to read as instructions; put the whole message in the chat.
- Open every message with `AGENT MESSAGE (<your tab label>, not a human).`
- One ask per message, in a few sentences. A long paste can collapse into a block the receiver drops.
- Ask for a plain-text reply with a line limit and no markdown.
- Treat the reply as the other agent's claim, not as verified fact.

## Steps

1. Find the target with `herdr tab list --workspace <id>` and `herdr agent list`. Use the agent's name, or the pane ID when it has no name.
2. Wait until it is not working: `herdr agent wait <target> --timeout 120000`.
3. If the target is a `claude` agent, set aside any draft the user left in its input box: `herdr agent send-keys <target> ctrl+s`. Claude puts the draft back after your message is sent.
4. Send and wait for the reply: `herdr agent prompt <target> "<message>" --wait --timeout 300000`.
5. Read the reply: `herdr agent read <target> --source recent-unwrapped --lines 80`. If it scrolled off, ask the tab to resend it shorter.
6. To discuss, repeat steps 3 to 5 until the question is settled, then report the outcome to the user.
