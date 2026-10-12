---
name: no-slop-writing
description: Draft or rewrite public prose using the author's actual writing and current intent as the voice reference. Use for blog posts, essays, technical notes, opinion pieces, and public-writing revisions.
---

# No-Slop Writing

Draft or revise public prose so the reader understands the subject and hears the author. Ground claims in the available facts and choose the structure from the piece's purpose, rather than imposing a house style.

## Local file

`~/.agents/local/no-slop-writing.md` holds excerpts of the user's own prose, with their source and limits. Read it when it exists and the piece is the user's own writing; without it, use the samples the user supplies.

## Scope

Use this skill for:

- Blog posts
- Engineering essays
- Technical notes
- Opinion pieces
- Draft rewrites
- Turning findings into readable writing

## Establish the voice

- Read the author's actual prose before drafting. Prefer supplied examples and relevant long-form work over a summary of style traits. Read the existing article when revising it.
- Current instructions and feedback take priority over inferred historical patterns. Distinguish preferences the author has stated from tendencies observed in a small sample.
- Agent-chat brevity, workplace replies, and public articles serve different purposes. Use chat to understand intent; do not automatically copy its compression or formatting into an article.
- When writing for the user, apply the local file's excerpts within their stated limits. Supplement them with relevant current or long-form samples when available. For other authors, use their own samples; do not transfer the user's voice to them.
- If no author samples are available, preserve the existing draft's voice and state the limitation when a voice match is requested. Do not claim to have reproduced a voice from generic writing advice alone.

## Structure and titles

Choose an opening that gives the reader a useful reason to continue: context, a question, a personal observation, a technical explanation, or an incident can each work. A scene is useful when it belongs to the material, but it is not compulsory.

Build the sequence around what the reader needs to understand. A failure report may move from setup to expectation, outcome, mechanism, and implication; an explanatory article may introduce the question and work through an example. Do not force every piece into a mismatch or a lesson about review time.

Choose a title that accurately names the subject or gives the reader a reason to investigate it. Do not require a two-sentence contrast, manufacture a surprising failure, or reject an informative title merely because it states an idea.

## Preserve what makes the prose personal

- Keep context and examples that explain the author's reasoning. Removing repeated words is useful; removing the explanation to make the prose terse can change the voice and make the argument harder to follow.
- Preserve meaningful parenthetical asides, qualifications, reader questions, and candid opinions when they belong to the author and the piece. Let longer explanatory sentences coexist with short ones instead of turning every paragraph into punchlines.
- Use connected paragraphs and the publication's capitalization and punctuation conventions. Casual chat spelling is evidence of register, not a requirement for edited prose.
- Do not insert a quota of catchphrases, hedges, ellipses, jokes, typos, or rhetorical questions. Voice comes from how the author develops a thought, not from sprinkling recognizable tics over it.
- Do not invent personal motivations, memories, feelings, dialogue, or facts to make a draft sound intimate. Attribute opinions to the author only when the source or current instructions support them.

## Edit with specificity

- Prefer concrete subjects, actions, examples, and mechanisms over stock praise or abstract claims. Explain what changed and why the evidence supports it.
- Remove filler, unsupported certainty, and repetitive phrasing. Size the language to the actual stakes and preserve uncertainty where the evidence is incomplete.
- Use comparisons when they clarify a real distinction. Avoid inserting stock contrast templates or manufactured trade-offs.
- Let the ending follow from the piece's actual finding, question, or implication. A slogan or universal rule is not required.
- Generic cleanup, including another editing skill, must preserve the author's reasoning and voice. If an edit makes the prose cleaner but removes a meaningful aside, qualification, or explanatory rhythm, reconsider the edit.

## Check

Before final output:

- Does the draft follow the author's current intent and fit the audience?
- Did actual prose inform the voice, with genre and age differences accounted for?
- Can the reader follow the reasoning, examples, and consequences without missing context?
- Are personal statements and factual claims supported?
- Did editing preserve useful asides, qualifications, and varied sentence rhythm?
- Do the opening, title, and ending serve this particular piece without a forced formula?
