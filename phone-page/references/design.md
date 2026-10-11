# Phone-page design

Every phone page uses this dark system. A read page and a question page should feel like two views of the same product.

## Foundation

- Canvas: `#0e0f0c`.
- Card: `#191b16`.
- Text: `#e8ebe6`.
- Muted text: `#a3a9a0`.
- Line: `#2c3029`.
- Lime: `#9fe870`, with `#0e0f0c` text on lime controls.
- Error: `#ffb4ab`.
- Headlines: Figtree at weight 900. Body: Inter at weights 400, 500 and 600.
- Cards have a 24 px radius. Buttons and compact labels are pills.
- Every foreground and background pair has at least 4.5:1 contrast. Muted text on a card is the lowest pair, at 7.2:1.

Use one column with a 16 px gutter on a phone. Keep that column centered and no wider than 720 px on a desktop browser. Controls are at least 44 px high, focus is visible, and the layout never requires horizontal scrolling.

## Content

- Before writing any page, use `git ls-files` in the current repository to find files named `glossary.md` at any path, ignoring letter case, and read any matches.
- Use glossary terms with their exact spelling and meaning, and define each term on first use using the glossary's own definition.

Open with one short eyebrow and one headline. State the bottom line in the first bullet, then give the reasons in the remaining bullets. Write each bullet and sub-bullet as one sentence at 17 px. Put numbered points in nested sub-bullets without numbering their text. Show every process as a flow chart with bordered boxes and arrows, using branches for decisions: use the question spec's `flow` field on a question page or draw the same chart on a read page.

Use Lucide icons only when they identify meaning. Pair every icon with visible text or an accessible name. Do not use an icon as decoration between paragraphs.

## Components

### Card

A card groups one decision, fact or outcome. Give it a short heading and bullet content, plus a small status chip when useful. Use the lime outline only for the current or selected card.

### Flow chart

Draw every process as bordered boxes joined by down arrows under a small Flow label, with 17 px text and the existing colors. A decision box connects by an arrow to each outcome box. Place outcomes side by side, or stack them beside a shared connector when the screen is too narrow; alternatives never appear as sequential steps. Use the same chart on read pages and question pages, never numbered circles or a code block.

In a question spec, `context` accepts one-sentence strings or objects `{"text": "<one sentence>", "items": ["<one sentence>", ...]}` whose non-empty `items` render as nested sub-bullets. `flow` accepts one-sentence strings or objects `{"text": "<one sentence decision>", "branches": ["<one sentence outcome>", ...]}` with 2 to 4 branches. Each bullet, sub-bullet, flow step, branch outcome and option description has one sentence. Validation rejects `.`, `!` or `?` followed by whitespace and an uppercase letter or digit, while allowing decimals and version numbers; bullets cannot start with `1.` or `2)` numbering.

### Options

An option is a full-width tappable card. Put its label first, its one-sentence consequence second, and the Recommended chip beside the first option when the spec marks it. Use radio buttons for one choice and checkboxes for multiple choices.

### Actions and status

The primary action is a lime pill. Put submission errors next to the action and move focus to the question that needs attention. After a successful submission, show its time and the exact instruction `Reply done in the chat`.

## Assets

Load scripts and fonts only from `/assets/<name>`, using the names in `../assets.json`. A page must not request a remote stylesheet, script, font, image, analytics endpoint or API. Inline page-specific CSS is allowed by the store's content policy.

Start a read page with these asset declarations:

```html
<style>
  @font-face { font-family: Figtree; font-style: normal; font-weight: 900; font-display: swap; src: url('/assets/figtree-900.woff2') format('woff2'); }
  @font-face { font-family: Inter; font-style: normal; font-weight: 400; font-display: swap; src: url('/assets/inter-400.woff2') format('woff2'); }
  @font-face { font-family: Inter; font-style: normal; font-weight: 500; font-display: swap; src: url('/assets/inter-500.woff2') format('woff2'); }
  @font-face { font-family: Inter; font-style: normal; font-weight: 600; font-display: swap; src: url('/assets/inter-600.woff2') format('woff2'); }
</style>
<script src="/assets/tailwind-browser.js"></script>
<script src="/assets/lucide.min.js"></script>
```
