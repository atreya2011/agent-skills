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

Use one column with a 16 px gutter on a phone. Read pages keep that column centered and no wider than 720 px on a desktop browser. Question pages below 960 px scroll through all questions; at 960 px and above each question starts a new screen and spans as many parts as needed, then comes a review screen, within a centered layout up to 1200 px wide. A question with context, cards or a table fills its left column, then its right column, with whole blocks in reading order: heading with context, cards, table, each chart, and options with Other, note and Skip. Other questions keep a centered single column. Measure blocks against the space between the fixed progress and navigation bars. A taller block owns a screen and scrolls internally; the desktop page does not scroll. Only the first screen holds the masthead; later parts begin with a compact question heading. Repack on resize while keeping the current question and any focused answer field. Controls are at least 44 px high, focus is visible, and the layout never requires horizontal scrolling.

## Content

- Before writing any page, use `git ls-files` in the current repository to find files named `glossary.md` at any path, ignoring letter case, and read any matches.
- Use glossary terms with their exact spelling and meaning, and define each term on first use using the glossary's own definition.

Open with one short eyebrow and one headline. State the bottom line in the first bullet, then give the reasons in the remaining bullets. Write each bullet and sub-bullet as one sentence at 17 px. Put numbered points in nested sub-bullets without numbering their text. Show every process as a flow chart with bordered boxes and arrows, using branches for decisions: use the question spec's `flow` or `flows` field on a question page or draw the same chart on a read page. Show parallel facts as a card grid and structured comparisons as a table; keep bullets for reasons that do not fit either.

- A page covers one project; questions or content about different projects go on separate pages; the page title names the project.

Use Lucide icons only when they identify meaning. Pair every icon with visible text or an accessible name. Do not use an icon as decoration between paragraphs.

## Components

### Summary

One sentence in text color at weight 500 directly under the headline states the bottom line. On a question page the spec's `summary` renders it; on a read page write it as the first paragraph.

### Card

A card groups one decision, fact or outcome. Give it a short heading and bullet content, plus a small status chip when useful. Use the lime outline only for the current or selected card.

### Fact cards

A grid of 2 to 6 small cards shows parallel facts such as causes, sources or constraints. Each has a canvas background inside the card, a 1 px line border, a 12 px radius, a 17 px title at weight 600 with an optional 20 px lime Lucide icon, and one 17 px muted sentence. The grid has one column on a phone and two columns when the container is at least 500 px wide. On a question page the spec's `cards` field renders it.

### Table

A table compares 2 to 4 columns over 1 to 8 rows at 17 px. Column headers are uppercase muted text at 0.75 rem, the first cell of each row is in text color at weight 500, other cells are muted, and rows are separated by a 1 px line. The table scrolls horizontally inside its wrapper instead of widening the page. On a question page the spec's `table` field renders it.

### Flow chart

Draw every process as bordered boxes joined by down arrows under a small label, with 17 px text and the existing colors. The label is `Flow` for a single chart, or the chart's title of up to 40 characters when a page shows more than one chart. A decision box connects by an arrow to each outcome box. Place outcomes side by side, or stack them beside a shared connector when the screen is too narrow; alternatives never appear as sequential steps. Use the same chart on read pages and question pages, never numbered circles or a code block.

In a question spec, `context` accepts one-sentence strings or objects `{"text": "<one sentence>", "items": ["<one sentence>", ...]}` whose non-empty `items` render as nested sub-bullets. `flow` accepts one-sentence strings or objects `{"text": "<one sentence decision>", "branches": ["<one sentence outcome>", ...]}` with 2 to 4 branches, and `flows` accepts 1 to 4 objects `{"title": "<up to 40 characters>", "steps": [...]}` with the same steps. A question renders context, cards, table, charts and options in that order. Each bullet, sub-bullet, card title, card text, table cell, flow title, flow step, branch outcome and option description has one sentence. Validation rejects `.`, `!` or `?` followed by whitespace and an uppercase letter or digit, while allowing decimals and version numbers; bullets cannot start with `1.` or `2)` numbering.

### Options

An option is a full-width tappable card. Put its label first, its one-sentence consequence second, and the Recommended chip beside the first option when the spec marks it. Use radio buttons for one choice and checkboxes for multiple choices.

### Actions and status

The primary action is a lime pill. Desktop question screens have a fixed bottom bar with Previous, a position such as Question 1 of 2, part 1 of 2, and Next. The bar is disabled until questions exist. Previous is disabled on the first screen; Next becomes Review answers on the final part and is hidden on review. Left and Right arrow keys navigate unless focus is in a text field, radio, checkbox or table, or a modifier key is held. Screen changes scroll to the top and focus the heading; keys at either end leave focus and scrolling alone.

The desktop review heading is Check your answers; on a phone, a live Your answers panel follows the last question. Each row shows the question's header and question, selected option labels, Other text, Skipped or Not answered, and any note prefixed with Note:. Its Change button opens that question's first desktop part or scrolls to and focuses its heading on a phone. Submit appears only on desktop review, centered below the rows, and full width on a phone; Enter from a question opens review first. Put submission errors near the action and move focus to the question that needs attention; an empty Other response opens the desktop part containing its field and focuses it. Put the latest submission notice and any failure to check it in the masthead so they show when the page opens. After a successful submission, show `Sent at` with the local time, then the exact instruction `Reply done in the chat`, below Submit.

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
