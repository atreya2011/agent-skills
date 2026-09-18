---
name: where-are-we
description: Reconstruct how the current session started and where it stands now by parsing its own transcript, then report it as a short readable narrative (ASD-STE100 mechanics, blunt outcome-first voice). Invoke in a long or resumed session to catch up.
disable-model-invocation: true
---

Answer "how did this session start and where are we now?" as a short report I can read in one sitting. Do the forensic work in silence; the report carries only the synthesis.

## WORK (all of this stays behind the scenes)

- Locate this session's transcript: `<config root>/projects/<cwd slug>/<session-id>.jsonl`, where config root = `$CLAUDE_CONFIG_DIR` if set else `~/.claude`, and slug = cwd with "/" replaced by "-". If context hides the session id, take the most recently modified .jsonl and say you inferred it.
- Parse it programmatically. Keep the real user messages in order (type=="user", isMeta false or missing) and drop interruptions, hook echoes, command output, and "Caveat:" wrappers.
- Re-verify every "current" claim live (git log/status, gh pr/issue state, file mtimes) before you state it. Where live state contradicts the transcript, write one sentence that says so.

## STYLE — ASD-STE100 mechanics, my voice on top

- Write complete sentences in active voice with a named actor ("I opened the PR", "You approved the plan").
- Keep each sentence to 20 words and one idea. Keep each paragraph to 6 sentences.
- Use one term for one thing through the whole report.
- Choose the simple word: "start" over "initiate", "do" over "perform", "show" over "indicate". Contractions are fine.
- Talk to me like a blunt colleague: state facts and results straight, and let them stand on their own.
- Use my vocabulary as-is: PR, CI, cwd, branch names, repo names, exactly as I write them.
- Pair every number with its meaning in the same sentence ("3 PRs open, all waiting on you").
- Describe outcomes ("the PR is open and green"); translate any activity line ("I ran 7 subagents") into the outcome it produced.
- Use present tense for current state, past tense for what happened, and future tense only for truly scheduled actions.

## EDIT PASS — find and eliminate before sending

- Bare numbers: attach the meaning in the same sentence, or cut the number.
- Hedging, apologies, and politeness filler: cut them.
- Transcript statistics, ids, and timestamps: cut them; keep dates only where they matter.
- Tables and bullet fragments: rewrite as prose.
- Noun clusters longer than 3 words: break them with prepositions.
- Synonym switches: restore the one chosen term.

Target length: about 300 words.

## STRUCTURE

1. TLDR: one sentence — where we are and the one thing waiting on me.
2. How it started: two or three sentences on my first real request and the goal.
3. The story since: one short paragraph per major arc, in order.
4. Rules I gave you mid-session that still apply, one sentence each.
