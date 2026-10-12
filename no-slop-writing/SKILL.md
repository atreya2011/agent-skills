---
name: no-slop-writing
description: Draft or rewrite public prose, or edit existing prose to remove AI tells, using the author's actual writing and current intent as the voice reference. Use for blog posts, essays, technical notes, opinion pieces, public-writing revisions, and unslop or de-AI passes.
---

# No-Slop Writing

Draft or revise prose so the reader understands the subject and hears the author. Ground claims in the available facts and choose the structure from the piece's purpose, rather than imposing a house style.

## Local file

`~/.agents/local/no-slop-writing.md` holds excerpts of the user's own prose, with their source and limits. Read it when it exists and the piece is the user's own writing; without it, use the samples the user supplies.

## Modes

- Draft mode writes or rewrites blog posts, engineering essays, technical notes, opinion pieces, and findings turned into readable writing.
- Edit mode removes AI tells from existing prose and keeps its meaning and the author's voice.

## Establish the voice

- Read the author's actual prose before drafting. Prefer supplied examples and relevant long-form work over a summary of style traits. Read the existing article when revising it.
- Current instructions and feedback take priority over inferred historical patterns. Distinguish preferences the author has stated from tendencies observed in a small sample.
- When writing for the user, apply the local file's excerpts within their stated limits. Supplement them with relevant current or long-form samples when available. For other authors, use their own samples; do not transfer the user's voice to them.
- Read [the voice rules](references/voice.md) for chat versus article register, missing samples, and what to preserve.

## Draft mode

Read [the drafting rules](references/drafting.md) to choose the opening, sequence, title and ending, and to edit for specificity.

## Edit mode

1. Scan for the patterns in [the AI tells](references/ai-tells.md), and leave its false positives alone.
2. Rewrite. Preserve meaning, match intended tone.
3. Self-audit: "What makes this obviously AI generated?" Fix remaining tells.

## Both modes

- Do not invent personal motivations, memories, feelings, dialogue, or facts to make a draft sound intimate. Attribute opinions to the author only when the source or current instructions support them.
- Cleanup, including edit mode, must preserve the author's reasoning and voice. If an edit makes the prose cleaner but removes a meaningful aside, qualification, or explanatory rhythm, reconsider the edit.

## Check

Before final output:

- Does the draft follow the author's current intent and fit the audience?
- Did actual prose inform the voice, with genre and age differences accounted for?
- Can the reader follow the reasoning, examples, and consequences without missing context?
- Are personal statements and factual claims supported?
- Did editing preserve useful asides, qualifications, and varied sentence rhythm?
- Do the opening, title, and ending serve this particular piece without a forced formula?
