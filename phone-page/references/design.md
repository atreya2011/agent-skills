# Phone-page design

Read and question specs render through one template, stylesheet and shared block renderers in this dark system. Author read pages only through `phone-page read <spec>` and question pages through `phone-page ask <spec>`; both validate their content before publishing.

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

Use one column with a 16 px gutter below 960 px. At 960 px and above, a read page scrolls inside a centered container up to 1200 px wide, with a full-width masthead and one full-width row per section. The section heading and context occupy the left column, cards occupy the right column, and its table and charts span both columns below them. A context-only section stays in one centered column. Read pages have no progress bar or Previous and Next bar. Question pages below 960 px scroll through all questions; at 960 px and above each question starts a new screen and spans as many parts as needed, then comes a review screen, within a centered layout up to 1200 px wide. A question with context, cards or a table fills its left column, then its right column, in reading order: heading, context, cards, table, each chart, and options with Other, note and Skip. Bulleted lists split between top-level bullets to fill the remaining column space, then continue in the next column or screen with the same styling and indentation; each bullet keeps its whole text and sub-bullets. A screen with an empty right column keeps one centered column at the measured width; other questions keep a centered single column. Measure blocks against the space between the fixed progress and navigation bars. A single bullet taller than a column, or a taller block that is not a list, owns a screen and scrolls internally; the desktop page does not scroll. Only the first screen holds the masthead; later parts begin with a compact question heading. Repack on resize while keeping the current question and any focused answer field. Controls are at least 44 px high, focus is visible, and the layout never requires horizontal scrolling.

## Content

- Before writing any page, use `git ls-files` in the current repository to find files named `glossary.md` at any path, ignoring letter case, and read any matches.
- Use glossary terms with their exact spelling and meaning, and define each term on first use using the glossary's own definition.

Open with one short eyebrow and one headline. State the bottom line in the first bullet, then give the reasons in the remaining bullets. Write each bullet and sub-bullet as one sentence at 17 px. Put numbered points in nested sub-bullets without numbering their text. Show every process as a flow chart with bordered boxes and arrows, using branches for decisions: use the shared `flow` or `flows` field on either page type. Show parallel facts as a card grid and structured comparisons as a table; keep bullets for reasons that do not fit either. A read page follows the same density rule as a question: the bottom line with its reason, the facts in a table with numbers and sources, two or three limits, the process as a chart, and nothing else.

- A page covers one project; questions or content about different projects go on separate pages; the page title names the project.

Use Lucide icons only when they identify meaning. Pair every icon with visible text or an accessible name. Do not use an icon as decoration between paragraphs.

## Components

### Summary

One sentence in text color at weight 500 directly under the headline states the bottom line. Both page types render it from the spec's `summary`.

### Card

A card groups one decision, fact or outcome. Give it a short heading and bullet content, plus a small status chip when useful. Use the lime outline only for the current or selected card.

### Fact cards

A grid of small cards shows parallel facts such as causes, sources or constraints. Each has a canvas background inside the card, a 1 px line border, a 12 px radius, a 17 px title at weight 600 with an optional 20 px lime Lucide icon, and one 17 px muted sentence. The grid has one column on a phone and two columns when the container is at least 500 px wide. Both page types render it from `cards`.

### Table

A table compares structured facts at 17 px. Column headers are uppercase muted text at 0.75 rem, the first cell of each row is in text color at weight 500, other cells are muted, and rows are separated by a 1 px line. The table scrolls horizontally inside its wrapper instead of widening the page. Both page types render it from `table`.

### Flow chart

Draw every process as bordered boxes joined by down arrows under a small label, with 17 px text and the existing colors. The label is `Flow` for a single chart, or the chart's title when a page shows more than one chart. A decision box connects by an arrow to each outcome box. Place outcomes side by side, or stack them beside a shared connector when the screen is too narrow; alternatives never appear as sequential steps. Use the same chart on read pages and question pages, never numbered circles or a code block.

### Mermaid diagram

Draw a state machine, cycle, sequence or relationship that the flow chart cannot draw as a Mermaid diagram under the same small label, showing its title. The diagram keeps its natural width inside a bordered box in the page colors and Inter; a diagram wider than the screen scrolls inside its box, so the page never scrolls horizontally. The page loads Mermaid from `/assets/` only when the spec has a `mermaid` block, renders with strict security, and shows a render error as text in that diagram's box.

The shared field schemas and sentence rules are defined once in [blocks.md](blocks.md); [read-spec.md](read-spec.md) and [question-spec.md](question-spec.md) define each page type's own fields. A question renders context, cards, table, charts and options in that order. A read section uses the desktop row layout above and that same content order on a phone.

### Options

An option is a full-width tappable card. Put its label first, its one-sentence consequence second, and a Recommended chip beside each option the spec marks. Use radio buttons for one choice and checkboxes for multiple choices.

### Actions and status

The primary action is a lime pill. Desktop question screens have a fixed bottom bar with Previous, a position such as Question 1 of 2, part 1 of 2, and Next. The bar is disabled until questions exist. Previous is disabled on the first screen; Next becomes Review answers on the final part and is hidden on review. Left and Right arrow keys navigate unless focus is in a text field, radio, checkbox, table or diagram, or a modifier key is held. Screen changes scroll to the top and focus the heading; keys at either end leave focus and scrolling alone.

The desktop review heading is Check your answers; on a phone, a live Your answers panel follows the last question. Each row shows the question's header and question, selected option labels, Other text, Skipped or Not answered, and any note prefixed with Note:. Its Change button opens that question's first desktop part or scrolls to and focuses its heading on a phone. Submit appears only on desktop review, centered below the rows, and full width on a phone; Enter from a question opens review first. Put submission errors near the action and move focus to the question that needs attention; an empty selected Other response keeps its error, opens the desktop part containing its field and focuses it. Put the latest submission notice and any failure to check it in the masthead so they show when the page opens. After a successful submission, keep the success toast and show `Sent at` with the local time, then the exact instruction `Reply done in the chat`, below Submit.

## Assets

The template loads scripts and fonts only from `/assets/<name>`, using the names in [assets.json](../assets.json), and supplies the font declarations and inline CSS. Pages must not request remote stylesheets, scripts, fonts, images, analytics endpoints or APIs.

Interactivity uses the pinned official `@alpinejs/csp` build without changing the store's content policy. Its [CSP expression limits](https://alpinejs.dev/advanced/csp) exclude arrow functions, destructuring, template literals and global functions; use simple property and registered method bindings, with logic in the Alpine component. Inline scripts remain allowed, and `eval` remains disallowed.
