---
name: unslop
description: Edit prose to remove AI tells and restore a natural human voice. Invoke explicitly for an unslop or de-AI pass.
disable-model-invocation: true
---

# Unslop

Edit text to remove AI patterns and add human voice.

## Answer, don't perform

Put the answer in the first sentence: the decision, result, or bottom line
the reader asked for. Keep a later sentence only if it changes what the
reader does or knows. A one-sentence reply is complete.

Bad:
> two findings, and one that collapses the whole decision tree. first: you
> haven't sent your post. it's sitting dormant in the text field — no bytes
> have touched the wire. second, and arguably worse: you're attempting to
> cancel your post. flagging this, because it's the kind of mistake that
> drops genuine data. your call on how to proceed: ...

Good:
> You haven't posted yet. Save it as draft to finish later, or discard it?

Same information. The bad version narrates its own analysis, dramatizes
stakes, and invents detail the reader cannot verify, and the decision
arrives last. Draft as long as you need, then send only the cut. The reader
never sees the draft.

## Top priority: outcomes over inventories

Read this before any pattern below. When reporting completed work, keep only the number, count, or check that changes what the reader does next; treat everything else as filler. Name the outcome and one line of proof; cut the rest. A dump of every verification result reads as thoroughness to the writer and as word vomit to the reader.

## Process

1. Scan for the patterns below.
2. Rewrite. Preserve meaning, match intended tone.
3. Add soul (see next section).
4. Self-audit: "What makes this obviously AI generated?" Fix remaining tells.

## Adding soul

Removing patterns is half the job. Sterile, voiceless writing is just as obvious.

- **Have opinions.** React to facts instead of neutrally listing pros and cons.
- **Vary rhythm.** Short sentences. Then longer ones that take their time. Mix it up.
- **Acknowledge complexity.** "Impressive but also kind of unsettling" beats "impressive."
- **Use "I" when it fits.** First person reads as professional too.
- **Let some mess in.** Perfect structure feels algorithmic.
- **Be specific.** Say "there's something unsettling about agents churning away at 3am" rather than "this is concerning."

## Patterns to detect and fix

### Content

1. **Significance inflation.** "pivotal moment", "testament to", "evolving landscape", "setting the stage for", "indelible mark", "deeply rooted". Cut puffery, state what happened.
2. **Notability name-dropping.** Listing media outlets without context. Pick one, say what was said.
3. **Superficial -ing phrases.** "highlighting...", "ensuring...", "reflecting...", "showcasing...", "fostering...". Delete or expand with real sources.
4. **Promotional language.** "nestled", "vibrant", "breathtaking", "groundbreaking", "renowned", "stunning", "must-visit". Use neutral descriptions.
5. **Vague attributions.** "Experts believe", "Industry reports suggest", "Some critics argue". Name the source or delete.
6. **Formulaic challenges.** "Despite challenges... continues to thrive." Replace with specific facts.

### Language

7. **AI vocabulary.** Additionally, crucial, delve, enduring, enhance, fostering, garner, interplay, intricate, landscape (abstract), pivotal, showcase, tapestry (abstract), testament, underscore, vibrant. Replace with plain words.
8. **Copula avoidance.** "serves as", "stands as", "boasts", "features". Just say "is" or "has".
9. **Negative parallelisms.** "It's not just X, it's Y." State the point directly.
10. **Rule of three.** Forcing ideas into groups of three. Use the natural number.
11. **Synonym cycling.** Protagonist, main character, central figure, hero all in one paragraph. Pick one, repeat it.
12. **False ranges.** "from X to Y" where X and Y aren't on a meaningful scale. List topics directly.

### Style

13. **Em dash overuse.** Replace every em dash with a period or a comma, using only those two for separation. Parentheses, en dashes, and hyphen-as-dash substitutes are equally an AI tell, since reaching for parentheses instead just trades one tell for another. When a thought needs separation, end the sentence or use a comma.
14. **Colon overuse.** Colons are fine before a list or example; keep them to that role and state a mid-sentence connection directly instead. "If you're coming from traditional automation: instead of registering event handlers, you describe conditions" adds nothing with the colon. Rewrite so the point stands on its own using plain framing rather than a comparison setup. "Describing when the scheduler should fire works best as plain English." Same meaning, plain punctuation only.
15. **Boldface overuse.** Reserve bold for the proper noun or acronym that truly needs emphasis, and leave the rest unbolded.
16. **Inline-header lists.** The tell is a bold label and colon that restates the line: "**Performance:** Performance improved...". Convert those to prose. A bold lead-in that ends in a period, names the item, and is followed by genuinely new detail ("**Schema in TypeScript.** Tables live in one file.") is fine, a legitimate craft choice rather than a tell.
17. **Title case headings.** Use sentence case.
18. **Decorative emojis.** Remove from headings and bullets.
19. **Curly quotes.** Replace with straight quotes.

### Communication artifacts

20. **Chatbot phrases.** "I hope this helps!", "Let me know if...", "Of course!", "Certainly!", "Found the smoking gun!" Remove.
21. **Cutoff disclaimers.** "While specific details are limited..." Find sources or remove.
22. **Sycophantic tone.** "Great question! You're absolutely right!" Respond directly.

### Filler

23. **Filler phrases.** "In order to" becomes "To". "Due to the fact that" becomes "Because". "It is important to note that" gets deleted.
24. **Excessive hedging.** "could potentially possibly be argued that it might" becomes "may".
25. **Generic conclusions.** "The future looks bright." State specific plans or facts.

### Jargon

26. **Abstract metaphor nouns.** Substrate, wedge, vector, locus, vantage, nexus, primitive (as noun), harness (as metaphor), surface (as in "API surface"), bedrock, scaffolding (as metaphor), modality, paradigm. These read as technical but usually have a plainer concrete word. "Substrate" becomes "base". "Wedge in" becomes "add". "Vector" becomes "way" or "method". Pick the concrete word.

### Plain speech

27. **Say the concrete thing.** State what something does, in mechanism or number terms, instead of wrapping the point in abstract framing or describing how it feels. "the database stays close at hand", "SQL you can read", "types that follow your schema" name a feeling. The fix names the mechanism or a number: "`.toSQL()` returns the exact string sent to the database", "a column rename fails the build". Ask what the sentence tells the reader to do or know, then write that. Restate it as a concrete instruction, fact, or number; cut whatever resists that restatement.
28. **Shorten or split dense sentences.** If the reader has to backtrack to parse a sentence, break it in two or drop clauses. One idea per sentence.
29. **Active voice.** Prefer it. Catch "is/are/was/were + past participle" and name the actor: "queries are validated" becomes "the compiler validates queries", "the file is parsed by the loader" becomes "the loader parses the file". Passive is fine only when the actor is unknown or genuinely irrelevant.
30. **Cut adverbs, or use a stronger verb.** "runs quickly" becomes "is fast" or the number. "significantly improves" becomes the measured delta. An adverb propping up a weak verb means the verb is wrong.
31. **Prefer the plain word.** "utilize" becomes "use", "leverage" becomes "use", "facilitate" becomes "help", "numerous" becomes "many", "in the event that" becomes "if". The fancier synonym is rarely clearer.
32. **Use clear, plain and literal language.** Name concrete things with plain, literal words instead of an invented metaphor. A filename that matches OCR is "confirmed by the filename", using the plain phrase in place of the metaphor "witness". If an invented term needs a definition to be understood, define it at first use or replace it with the plain phrase.
33. **Outcomes, not inventories.** See "Top priority" at the head of this file; it applies to every report of completed work.

## Leave these alone (false positives)

A clean human writer can hit several patterns above with no AI involved. Preserve legitimate prose across these cases:

- Perfect grammar and consistent style. Professionals get edited too.
- Mixed casual and formal registers. Often a technical writer or a young writer, not a chatbot.
- "Bland" or "robotic" prose alone. AI tells are specific; generic dryness isn't one of them.
- Formal or academic vocabulary in general. Only the specific words in #7 are AI-coded.
- Letter-style openings or sign-offs. Predate chatbots by centuries.
- A single transition word (however, additionally, moreover). Only a tell when piled up.
- Curly quotes alone. Most editors and CMSes auto-curl by default.
- A single em dash. Evidence only paired with formulaic, sales-y rhythm.
- Unsourced claims alone. Most of the web is unsourced; that proves nothing.
- Clean, correct formatting from a template or visual editor.

Look for clusters of tells rather than isolated ones. One em dash means nothing; em dash plus rule-of-three plus "vibrant tapestry" plus a "Conclusion" section is a confession.

## Signs of human writing (preserve these)

Lean toward leaving the prose alone when you see these. Over-editing destroys what makes it sound human:

- Specific, unusual, hard-to-fabricate detail. A real address, a weird quote. AI rounds off specifics; humans hoard them.
- Mixed feelings and unresolved tension. AI defaults to clean, resolved takes.
- Dated, era-bound slang or in-jokes. Models lag behind culture by a year or more.
- First-person editorial choices the writer can defend, with a reason.
- Real variety in sentence length, not an even mid-length cadence.
- Genuine asides, parentheticals, or self-corrections mid-thought.
- Edits from before ChatGPT's November 2022 launch. Reliably not AI-written.

## Reference

Patterns adapted from [Wikipedia:Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing), maintained by WikiProject AI Cleanup.
