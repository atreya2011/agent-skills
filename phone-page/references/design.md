# Phone-page design

Every phone page uses this dark system. A read page and a question page should feel like two views of the same product.

## Foundation

- Canvas: `#0e0f0c`.
- Card: `#191b16`.
- Text: `#e8ebe6`.
- Muted text: `#a3a9a0`.
- Line: `#2c3029`.
- Lime: `#9fe870`, with `#0e0f0c` text on lime controls.
- Headlines: Figtree at weight 900. Body: Inter at weights 400, 500 and 600.
- Cards have a 24 px radius. Buttons and compact labels are pills.
- Every foreground and background pair has at least 4.5:1 contrast. Muted text on a card is the lowest pair, at 7.2:1.

Use one column with a 16 px gutter on a phone. Keep that column centered and no wider than 720 px on a desktop browser. Controls are at least 44 px high, focus is visible, and the layout never requires horizontal scrolling.

## Content

Open with one short eyebrow, one headline and one sentence that says what the page is for. Each card or step explains one item in one to three full sentences. Do not use fragments. Split longer content into more cards, a short list, or a step flow.

Use Lucide icons only when they identify meaning. Pair every icon with visible text or an accessible name. Do not use an icon as decoration between paragraphs.

## Components

### Card

A card groups one decision, fact or outcome. Give it a short heading, a complete explanation and, when useful, a small status chip. Use the lime outline only for the current or selected card.

### Step flow

Draw a process as a vertical icon stepper. Each step has a numbered or semantic icon, a title and one to three full sentences. Use a line between steps; never put a flow in a code block.

### Options

An option is a full-width tappable card. Put its label first, its one-sentence consequence second, and the Recommended chip beside the first option when the spec marks it. Use radio buttons for one choice and checkboxes for multiple choices.

### Actions and status

The primary action is a lime pill. Put submission errors next to the action and move focus to the question that needs attention. After a successful submission, show its time and the exact instruction `Reply done in the chat`.

## Assets

Load scripts and fonts only from `/assets/<name>`, using the names in `../assets.json`. A page must not request a remote stylesheet, script, font, image, analytics endpoint or API. Inline page-specific CSS is allowed by the store's content policy.
