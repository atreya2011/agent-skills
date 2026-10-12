---
name: herdr-chat
description: Talk to another agent through its herdr tab, by chat only. Use when the user asks to discuss with, ask, brief or check on another tab or agent.
---

# herdr-chat

Chat is the only channel to another agent's tab. This skill owns message format, reply handling and busy-tab behavior; [herdr-orchestrator](../herdr-orchestrator/SKILL.md) owns tab lifecycle and CLI syntax.

## Rules

- Never edit the other tab's files. Never hand it a file to read as instructions; put the whole message in the chat.
- Open every message with `AGENT MESSAGE (<your tab label>, not a human).`
- One ask per message, in a few sentences. A long paste can collapse into a block the receiver drops.
- Ask for a plain-text reply with a line limit and no markdown.
- Treat the reply as the other agent's claim, not as verified fact.

## Steps

1. Read [herdr-orchestrator/references/herdr-cli.md §Lists and targets](../herdr-orchestrator/references/herdr-cli.md#lists-and-targets) and [§Agent commands](../herdr-orchestrator/references/herdr-cli.md#agent-commands) before finding a target or using prompt, wait, send-keys or read. Find the target in its workspace; use the owner's targeting rules.
2. Wait until it is not working, with a 120000 ms timeout.
3. If the target is a `claude` agent, send the logical key `ctrl+s` to set aside any user draft. Claude restores it after your message is sent.
4. Prompt the target with the whole message and wait for the reply, with a 300000 ms timeout.
5. Read the reply from `recent-unwrapped`, with 80 lines. If it scrolled off, ask the tab to resend it shorter.
6. To discuss, repeat steps 3 to 5 until the question is settled, then report the outcome to the user.
