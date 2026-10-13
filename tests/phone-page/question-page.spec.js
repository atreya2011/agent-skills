import { $ } from 'bun';
import { mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

const origin = 'https://phone-page.example';
const pageUrl = `${origin}/p/questions`;
const testDirectory = import.meta.dir;
const root = resolve(testDirectory, '../..');
const command = join(root, 'phone-page/phone-page');

const spec = {
  title: 'Choose the next move',
  questions: [
    {
      id: 'route',
      header: 'Route',
      question: 'Which route should the work take?',
      context: [
        'Take the direct route.',
        'It is ready without another dependency.'
      ],
      flow: [
        'Choose the route.',
        'Run the checks.',
        'Deliver the result.'
      ],
      multiSelect: false,
      options: [
        { label: 'Direct', description: 'Use the ready route with no added dependency.', recommended: true },
        { label: 'Separate', description: 'Create a separate route with its own lifecycle.' }
      ]
    },
    {
      id: 'delivery',
      header: 'Delivery',
      question: 'How should the result be delivered?',
      multiSelect: false,
      options: [
        { label: 'Ship', description: 'Deliver the completed result now.', recommended: true },
        { label: 'Hold', description: 'Keep the result ready without delivering it.' }
      ]
    },
    {
      id: 'checks',
      header: 'Checks',
      question: 'Which checks should run?',
      multiSelect: true,
      options: [
        { label: 'Behavior', description: 'Check the behavior a user can see.', recommended: true },
        { label: 'Access', description: 'Check keyboard and screen-reader access.', recommended: true },
        { label: 'Layout', description: 'Check the phone and desktop layouts.' }
      ]
    },
    {
      id: 'later',
      header: 'Later',
      question: 'Which follow-up should happen later?',
      multiSelect: false,
      options: [
        { label: 'Document', description: 'Write a separate follow-up document.', recommended: true },
        { label: 'Measure', description: 'Collect measurements before deciding.' }
      ]
    }
  ]
};

let renderedPage;
let renderDirectory;

test.beforeAll(async () => {
  renderDirectory = await mkdtemp(join(tmpdir(), 'phone-page-render-'));
  const specPath = join(renderDirectory, 'spec.json');
  await Bun.write(specPath, JSON.stringify(spec));
  renderedPage = await $`${command} render ${specPath}`.text();
});

test.afterAll(async () => {
  await $`rm -rf ${renderDirectory}`;
});

test('answers work on a phone and the page remains accessible', async ({ page }) => {
  const assetsDirectory = process.env.PHONE_PAGE_ASSETS_DIR;
  expect(assetsDirectory).toBeTruthy();
  const externalRequests = [];
  let latestSubmission;
  let failNextSubmission = false;
  const postedSubmissions = [];

  await page.route('**/*', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.origin !== origin) {
      externalRequests.push(url.href);
      await route.abort();
      return;
    }

    if (url.href === pageUrl && request.method() === 'GET') {
      await route.fulfill({ status: 200, contentType: 'text/html', body: renderedPage });
      return;
    }

    if (url.pathname.startsWith('/assets/') && request.method() === 'GET') {
      const name = (await $`basename ${url.pathname}`.text()).trimEnd();
      const contentTypes = {
        '.js': 'text/javascript',
        '.woff2': 'font/woff2'
      };
      const extension = name.endsWith('.woff2') ? '.woff2' : '.js';
      await route.fulfill({
        status: 200,
        contentType: contentTypes[extension],
        body: Buffer.from(await Bun.file(join(assetsDirectory, name)).bytes())
      });
      return;
    }

    if (url.href === `${pageUrl}/answers` && request.method() === 'GET') {
      if (!latestSubmission) {
        await route.fulfill({ status: 404, contentType: 'application/json', body: '{}' });
      } else {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(latestSubmission) });
      }
      return;
    }

    if (url.href === `${pageUrl}/answers` && request.method() === 'POST') {
      if (failNextSubmission) {
        failNextSubmission = false;
        await route.fulfill({ status: 503 });
        return;
      }
      latestSubmission = request.postDataJSON();
      postedSubmissions.push(latestSubmission);
      await route.fulfill({ status: 204 });
      return;
    }

    await route.fulfill({ status: 404, body: 'not found' });
  });

  await page.setViewportSize({ width: 390, height: 844 });
  expect(page.viewportSize()).toEqual({ width: 390, height: 844 });
  await page.goto(pageUrl);
  await expect(page.getByRole('heading', { level: 1, name: spec.title })).toBeVisible();

  const questionCards = page.locator('.question-card');
  await expect(questionCards).toHaveCount(spec.questions.length);
  for (let index = 0; index < spec.questions.length; index += 1) {
    await expect(questionCards.nth(index).getByLabel('Other', { exact: true })).toBeVisible();
    await expect(questionCards.nth(index).getByLabel('Note (optional)')).toBeVisible();
  }

  const routeQuestion = page.locator('[data-question-id="route"]');
  await expect(routeQuestion.locator('.context')).toHaveRole('list');
  await expect(routeQuestion.locator('.context li')).toHaveText(spec.questions[0].context);
  await expect(routeQuestion.locator('.flow-label')).toHaveText('Flow');
  await expect(routeQuestion.locator('.flow-box')).toHaveText(spec.questions[0].flow);
  await expect(routeQuestion.locator('.flow > .flow-arrow')).toHaveText(['↓', '↓']);
  await expect(routeQuestion.locator('.flow')).toBeVisible();
  expect(await routeQuestion.evaluate((question) => {
    const context = question.querySelector('.context');
    const flow = question.querySelector('.flow');
    return Boolean(context.compareDocumentPosition(flow) & Node.DOCUMENT_POSITION_FOLLOWING)
      && flow.getBoundingClientRect().top >= context.getBoundingClientRect().bottom;
  })).toBe(true);
  await expect(routeQuestion.locator('.flow + .answer-controls > .options')).toBeVisible();
  expect(await routeQuestion.locator('.context li').first().evaluate((element) => getComputedStyle(element).fontSize)).toBe('17px');
  expect(await routeQuestion.locator('.flow-box').first().evaluate((element) => getComputedStyle(element).fontSize)).toBe('17px');
  expect(await routeQuestion.locator('.flow-box').evaluateAll((boxes) => boxes.map((box) => getComputedStyle(box).borderStyle))).toEqual(['solid', 'solid', 'solid']);
  await routeQuestion.getByLabel('Other', { exact: true }).check();
  await expect(routeQuestion.getByLabel('Other response')).toBeVisible();
  await page.getByRole('button', { name: 'Submit answers' }).click();
  await expect(page.getByRole('alert')).toContainText('Write the Other response');
  expect(postedSubmissions).toHaveLength(0);

  await routeQuestion.getByLabel('Other response').fill('Use the staged route.');
  const deliveryQuestion = page.locator('[data-question-id="delivery"]');
  await deliveryQuestion.getByLabel('Ship', { exact: true }).check();
  await deliveryQuestion.getByLabel('Note (optional)').fill('Send the result in the current thread.');
  const checksQuestion = page.locator('[data-question-id="checks"]');
  await expect(checksQuestion.locator('.option:has(.chip) .option-label')).toHaveText(['Behavior', 'Access']);
  await expect(checksQuestion.locator('input[data-choice]')).toHaveCount(3);
  expect(await checksQuestion.locator('input[data-choice]').evaluateAll((inputs) => inputs.map((input) => input.type))).toEqual(['checkbox', 'checkbox', 'checkbox']);
  await checksQuestion.getByLabel('Behavior', { exact: true }).check();
  await checksQuestion.getByLabel('Access', { exact: true }).check();

  await page.reload();
  await expect(routeQuestion.getByLabel('Other', { exact: true })).toBeChecked();
  await expect(routeQuestion.getByLabel('Other response')).toHaveValue('Use the staged route.');
  await expect(deliveryQuestion.getByLabel('Ship', { exact: true })).toBeChecked();
  await expect(deliveryQuestion.getByLabel('Note (optional)')).toHaveValue('Send the result in the current thread.');
  await expect(checksQuestion.getByLabel('Behavior', { exact: true })).toBeChecked();
  await expect(checksQuestion.getByLabel('Access', { exact: true })).toBeChecked();
  await expect(page.getByText('3 of 4 answered')).toBeVisible();

  await deliveryQuestion.getByRole('button', { name: 'Skip this question' }).click();
  await checksQuestion.getByRole('button', { name: 'Skip this question' }).click();
  const laterQuestion = page.locator('[data-question-id="later"]');
  await laterQuestion.getByRole('button', { name: 'Skip this question' }).click();
  await page.getByRole('button', { name: 'Submit answers' }).click();
  await expect(page.getByRole('status')).toHaveText('Answers sent');
  await expect(page.getByRole('status').locator('.lucide-check-circle-2')).toBeVisible();
  await expect(page.locator('#reply')).toBeVisible();
  await expect(page.locator('#reply')).toContainText('Reply done in the chat');
  expect(postedSubmissions).toHaveLength(1);
  expect(postedSubmissions[0].answers[0]).toMatchObject({
    id: 'route',
    choices: [],
    other: 'Use the staged route.'
  });

  failNextSubmission = true;
  await page.getByRole('button', { name: 'Submit answers' }).click();
  await expect(page.getByRole('alert')).toHaveText('The answers were not sent. Check the connection and submit again.');
  await expect(page.getByRole('alert').locator('.lucide-circle-alert')).toBeVisible();
  await expect(page.locator('#form-error')).toBeHidden();
  expect(postedSubmissions).toHaveLength(1);

  await page.reload();
  await expect(page.getByText('Latest submission:')).toBeVisible();
  await expect(routeQuestion.getByLabel('Other', { exact: true })).toBeChecked();
  await expect(routeQuestion.getByLabel('Other response')).toHaveValue('Use the staged route.');
  await deliveryQuestion.getByLabel('Ship', { exact: true }).check();
  await deliveryQuestion.getByLabel('Note (optional)').fill('Send the result in the current thread.');
  await checksQuestion.getByLabel('Behavior', { exact: true }).check();
  await checksQuestion.getByLabel('Access', { exact: true }).check();
  await laterQuestion.getByRole('button', { name: 'Skip this question' }).click();
  await page.getByRole('button', { name: 'Submit answers' }).click();
  await expect(page.getByRole('status')).toHaveText('Answers sent');
  await expect(page.locator('#reply')).toBeVisible();
  await expect(page.locator('#reply')).toContainText('Reply done in the chat');

  expect(postedSubmissions).toHaveLength(2);
  const postedSubmission = postedSubmissions[1];
  expect(postedSubmission.submittedAt).toMatch(/^\d{4}-\d{2}-\d{2}T/);
  expect(postedSubmission.answers).toEqual([
    {
      id: 'route',
      header: 'Route',
      question: 'Which route should the work take?',
      choices: [],
      other: 'Use the staged route.',
      note: ''
    },
    {
      id: 'delivery',
      header: 'Delivery',
      question: 'How should the result be delivered?',
      choices: ['Ship'],
      other: '',
      note: 'Send the result in the current thread.'
    },
    {
      id: 'checks',
      header: 'Checks',
      question: 'Which checks should run?',
      choices: ['Behavior', 'Access'],
      other: '',
      note: ''
    },
    {
      id: 'later',
      header: 'Later',
      question: 'Which follow-up should happen later?',
      choices: [],
      other: '',
      note: ''
    }
  ]);

  await page.reload();
  await expect(page.getByText('Latest submission:')).toBeVisible();
  expect(externalRequests).toEqual([]);

  const mobileOverflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(mobileOverflow).toBeLessThanOrEqual(0);
  const mobileAccessibility = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  expect(mobileAccessibility.violations).toEqual([]);

  await page.setViewportSize({ width: 1280, height: 900 });
  const desktopOverflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(desktopOverflow).toBeLessThanOrEqual(0);
  const desktopAccessibility = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
  expect(desktopAccessibility.violations).toEqual([]);
});

for (const branchCount of [2, 4]) {
  test(`nested bullets and ${branchCount} flow branches render as a responsive chart`, async ({ page }) => {
    const question = structuredClone(spec.questions[0]);
    const items = ['Use the existing path.', 'Keep its lifecycle.'];
    const branches = ['Deliver the result.', 'Repair the result.', 'Review the result.', 'Hold the result.'].slice(0, branchCount);
    question.context = ['Take the direct route.', { text: 'Keep the change small.', items }];
    question.flow = ['Run the checks.', { text: 'Did the checks pass?', branches }];
    const specPath = join(renderDirectory, `branches-${branchCount}.json`);
    await Bun.write(specPath, JSON.stringify({ title: 'Choose a route', questions: [question] }));
    const html = await $`${command} render ${specPath}`.text();
    const server = Bun.serve({
      hostname: 'localhost',
      port: 0,
      async fetch(request) {
        const pathname = new URL(request.url).pathname;
        if (pathname === '/') {
          return new Response(html, { headers: { 'Content-Type': 'text/html' } });
        } else if (pathname.startsWith('/assets/')) {
          const name = (await $`basename ${pathname}`.text()).trimEnd();
          return new Response(Bun.file(join(process.env.PHONE_PAGE_ASSETS_DIR, name)), {
            headers: { 'Content-Type': name.endsWith('.woff2') ? 'font/woff2' : 'text/javascript' }
          });
        }
        return new Response(null, { status: 404 });
      }
    });
    try {
      for (const width of [420, 900]) {
        await page.setViewportSize({ width, height: 1100 });
        await page.goto(`http://localhost:${server.port}/`);
        const card = page.locator('.question-card');
        await expect(card.locator('.context > li')).toHaveCount(2);
        await expect(card.locator('.context > li').last()).toContainText('Keep the change small.');
        await expect(card.locator('.context > li > ul')).toHaveRole('list');
        await expect(card.locator('.context > li > ul > li')).toHaveText(items);
        await expect(card.locator('.flow-label')).toHaveText('Flow');
        await expect(card.locator('.flow-decision')).toHaveText('Did the checks pass?');
        await expect(card.locator('.flow-branch > .flow-box')).toHaveText(branches);
        await expect(card.locator('.flow-branch > .flow-arrow')).toHaveCount(branchCount);
        await expect(card.locator('.flow > .flow-arrow')).toHaveText('↓');
        await expect(card.locator('.flow-number')).toHaveCount(0);
        expect(await card.locator('.flow-box').evaluateAll((boxes) => boxes.map((box) => getComputedStyle(box).fontSize))).toEqual(Array(branchCount + 2).fill('17px'));
        const positions = await card.locator('.flow-branch').evaluateAll((elements) => elements.map((element) => {
          const { x, y } = element.getBoundingClientRect();
          return { x, y };
        }));
        if (width === 420) {
          expect(new Set(positions.map(({ x }) => x)).size).toBe(1);
          expect(new Set(positions.map(({ y }) => y)).size).toBe(branchCount);
        } else {
          expect(new Set(positions.map(({ x }) => x)).size).toBe(branchCount);
          expect(new Set(positions.map(({ y }) => y)).size).toBe(1);
        }
        expect(await card.locator('.flow-branch > .flow-arrow').evaluateAll((arrows) => arrows.map((arrow) => getComputedStyle(arrow, '::before').content))).toEqual(Array(branchCount).fill(width === 420 ? '"→"' : '"↓"'));
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
      }
    } finally {
      await server.stop(true);
    }
  });
}

for (const { name, width } of [
  { name: 'phone', width: 420 },
  { name: 'desktop', width: 900 }
]) {
  test(`summary, cards, table and named flows render on ${name}`, async ({ page }, testInfo) => {
    const question = structuredClone(spec.questions[0]);
    delete question.flow;
    question.cards = [
      { title: 'Ready route', text: 'Use the existing path.', icon: 'check' },
      { title: 'Separate route', text: 'Create another lifecycle.' }
    ];
    question.table = {
      columns: ['Route', 'Status'],
      rows: [['Direct', 'Ready'], ['Separate', 'Needs setup']]
    };
    question.flows = [
      { title: 'Direct route', steps: ['Run the checks.', 'Deliver the result.'] },
      { title: 'Separate route', steps: ['Create the path.', 'Run the checks.', 'Deliver the result.'] }
    ];
    const summary = 'Choose a route after comparing its readiness and steps.';
    const specPath = join(renderDirectory, `content-${name}.json`);
    await Bun.write(specPath, JSON.stringify({ title: spec.title, summary, questions: [question] }));
    const html = await $`${command} render ${specPath}`.text();
    const server = Bun.serve({
      hostname: 'localhost',
      port: 0,
      async fetch(request) {
        const pathname = new URL(request.url).pathname;
        if (pathname === '/') {
          return new Response(html, { headers: { 'Content-Type': 'text/html' } });
        } else if (pathname.startsWith('/assets/')) {
          const assetName = (await $`basename ${pathname}`.text()).trimEnd();
          return new Response(Bun.file(join(process.env.PHONE_PAGE_ASSETS_DIR, assetName)), {
            headers: { 'Content-Type': assetName.endsWith('.woff2') ? 'font/woff2' : 'text/javascript' }
          });
        }
        return new Response(null, { status: 404 });
      }
    });
    try {
      await page.setViewportSize({ width, height: 1100 });
      await page.goto(`http://localhost:${server.port}/`);
      await expect(page.locator('.intro.summary')).toHaveText(summary);
      const card = page.locator('.question-card');
      await expect(card.locator('.card-title')).toHaveText(question.cards.map(({ title }) => title));
      await expect(card.locator('.card-text')).toHaveText(question.cards.map(({ text }) => text));
      await expect(card.locator('.card-title .lucide-check')).toBeVisible();
      await expect(card.getByRole('table')).toBeVisible();
      await expect(card.locator('.table-wrap')).toHaveAttribute('tabindex', '0');
      await expect(card.getByRole('columnheader')).toHaveText(question.table.columns);
      await expect(card.getByRole('cell')).toHaveText(question.table.rows.flat());
      await expect(card.locator('.flow-label')).toHaveText(question.flows.map(({ title }) => title));
      for (const [index, flow] of question.flows.entries()) {
        await expect(card.locator('.flow').nth(index)).toBeVisible();
        await expect(card.locator('.flow').nth(index).locator('.flow-box')).toHaveText(flow.steps);
      }
      await page.screenshot({ path: testInfo.outputPath('content.png'), fullPage: true });
    } finally {
      await server.stop(true);
    }
  });
}

for (const { name, width, stepped } of [
  { name: 'phone', width: 420, stepped: false },
  { name: 'below desktop breakpoint', width: 959, stepped: false },
  { name: 'desktop breakpoint', width: 960, stepped: true },
  { name: 'wide desktop', width: 1280, stepped: true }
]) {
  test(`answer review and changes work on ${name}`, async ({ page }, testInfo) => {
    const server = Bun.serve({
      hostname: 'localhost',
      port: 0,
      async fetch(request) {
        const pathname = new URL(request.url).pathname;
        if (pathname === '/') {
          return new Response(renderedPage, { headers: { 'Content-Type': 'text/html' } });
        } else if (pathname.startsWith('/assets/')) {
          const assetName = (await $`basename ${pathname}`.text()).trimEnd();
          return new Response(Bun.file(join(process.env.PHONE_PAGE_ASSETS_DIR, assetName)), {
            headers: { 'Content-Type': assetName.endsWith('.woff2') ? 'font/woff2' : 'text/javascript' }
          });
        }
        return new Response(null, { status: 404 });
      }
    });
    try {
      console.log(`Open the ${name} form and choose Direct.`);
      await page.setViewportSize({ width, height: 1100 });
      await page.goto(`http://localhost:${server.port}/`);
      const cards = page.locator('.question-card');
      const panel = page.locator('#answer-panel');
      const rows = panel.locator('.answer-row');
      const submit = page.getByRole('button', { name: 'Submit answers' });
      await page.screenshot({ path: testInfo.outputPath('first-screen.png'), fullPage: true });
      if (stepped) {
        await expect(submit).toBeHidden();
        await expect(panel).toBeHidden();
        await expect(page.getByRole('button', { name: 'Previous', exact: true })).toBeDisabled();
        await expect(page.locator('#position')).toHaveText(/^Question 1 of 4(?:, part \d+ of \d+)?$/);
        const columns = await cards.nth(0).locator('.question-part:not([hidden]) .screen-column').evaluateAll((elements) => elements.map((element) => {
          const { x, y } = element.getBoundingClientRect();
          return { x, y };
        }));
        expect(columns).toHaveLength(2);
        expect(columns[1].x).toBeGreaterThan(columns[0].x);
        expect(columns[1].y).toBe(columns[0].y);
        const direct = cards.nth(0).getByLabel('Direct', { exact: true });
        for (let step = 0; step < 100 && !await direct.isVisible(); step += 1) await page.locator('#next').click();
        await direct.check();
        await page.screenshot({ path: testInfo.outputPath('first-answer.png'), fullPage: true });
        console.log('Walk Next through every question part to Review.');
        const answered = new Set([0]);
        for (let step = 0; step < 100 && !await panel.isVisible(); step += 1) {
          await page.locator('#next').click();
          await page.screenshot({ path: testInfo.outputPath(`screen-${step + 2}.png`), fullPage: true });
          if (await panel.isVisible()) break;
          const visibleCard = cards.filter({ visible: true });
          await expect(visibleCard).toHaveCount(1);
          await expect(visibleCard.getByRole('heading', { level: 2 })).toBeFocused();
          await expect(submit).toBeHidden();
          await expect(page.locator('.masthead')).toBeHidden();
          expect(await page.evaluate(() => window.scrollY)).toBe(0);
          const questionId = await visibleCard.getAttribute('data-question-id');
          const index = spec.questions.findIndex(({ id }) => id === questionId);
          if (answered.has(index)) continue;
          const control = index === 1 ? visibleCard.getByLabel('Ship', { exact: true })
            : index === 2 ? visibleCard.getByLabel('Behavior', { exact: true })
              : visibleCard.getByRole('button', { name: 'Skip this question' });
          if (!await control.isVisible()) continue;
          if (index === 1) {
            await control.check();
            await visibleCard.getByLabel('Note (optional)').fill('Send it here.');
          } else if (index === 2) {
            await control.check();
            await visibleCard.getByLabel('Access', { exact: true }).check();
          } else {
            await control.click();
          }
          answered.add(index);
        }
        expect([...answered].sort()).toEqual([0, 1, 2, 3]);
        await expect(page.getByRole('heading', { name: 'Check your answers' })).toBeFocused();
        await expect(cards.filter({ visible: true })).toHaveCount(0);
        await expect(page.locator('#position')).toHaveText('Review');
        await expect(page.locator('#next')).toBeHidden();
        await expect(submit).toBeVisible();
        const box = await submit.boundingBox();
        expect(Math.abs(box.x + box.width / 2 - width / 2)).toBeLessThanOrEqual(1);
        await expect(rows.nth(1)).toContainText('Note: Send it here.');
        await expect(rows.nth(2)).toContainText('Behavior');
        await expect(rows.nth(2)).toContainText('Access');
        await expect(rows.nth(3)).toContainText('Skipped');
      } else {
        await cards.nth(0).getByLabel('Direct', { exact: true }).check();
        await page.screenshot({ path: testInfo.outputPath('first-answer.png'), fullPage: true });
        await expect(page.getByRole('heading', { name: 'Your answers' })).toBeVisible();
        await expect(rows.nth(1)).toContainText('Not answered');
        await expect(submit).toBeVisible();
        await cards.nth(1).getByLabel('Ship', { exact: true }).check();
        await cards.nth(1).getByLabel('Note (optional)').fill('Send it here.');
        await expect(rows.nth(1)).toContainText('Ship');
        await expect(rows.nth(1)).toContainText('Note: Send it here.');
      }
      await expect(rows).toHaveCount(spec.questions.length);
      await expect(rows.nth(0)).toContainText('Route');
      await expect(rows.nth(0)).toContainText(spec.questions[0].question);
      await expect(rows.nth(0)).toContainText('Direct');
      await page.screenshot({ path: testInfo.outputPath('answer-review.png'), fullPage: true });

      console.log('Change Delivery and confirm the summary updates.');
      await rows.nth(1).getByRole('button', { name: 'Change Delivery' }).click();
      await expect(cards.nth(1).getByRole('heading', { level: 2 })).toBeFocused();
      await cards.nth(1).getByLabel('Hold', { exact: true }).check();
      if (stepped) {
        await expect(page.locator('#position')).toHaveText(/^Question 2 of 4(?:, part \d+ of \d+)?$/);
        await expect(submit).toBeHidden();
        await cards.nth(1).getByLabel('Note (optional)').focus();
        await page.keyboard.press('ArrowRight');
        await expect(page.locator('#position')).toHaveText(/^Question 2 of 4(?:, part \d+ of \d+)?$/);
        await cards.nth(1).getByRole('heading', { level: 2 }).focus();
        await page.keyboard.press('ArrowLeft');
        await expect(cards.nth(0).getByRole('heading', { level: 2 })).toBeFocused();
        for (let step = 0; step < 100 && !await page.locator('#previous').isDisabled(); step += 1) await page.keyboard.press('ArrowLeft');
        await expect(page.locator('.masthead')).toBeVisible();
        await expect(cards.nth(0).locator('h1:focus, h2:focus')).toHaveCount(1);
        await page.keyboard.press('ArrowRight');
        for (let step = 0; step < 100 && !await panel.isVisible(); step += 1) await page.keyboard.press('ArrowRight');
        await expect(panel).toBeVisible();
      }
      await expect(rows.nth(1)).toContainText('Hold');
      await expect(rows.nth(1)).not.toContainText('Ship');
      await page.screenshot({ path: testInfo.outputPath('changed-answer.png'), fullPage: true });

      console.log('Select Other without text and verify submission returns to its field.');
      await rows.nth(0).getByRole('button', { name: 'Change Route' }).click();
      const other = cards.nth(0).getByLabel('Other', { exact: true });
      if (stepped) {
        for (let step = 0; step < 100 && !await other.isVisible(); step += 1) await page.locator('#next').click();
      }
      await other.check();
      if (stepped) {
        await cards.nth(0).getByRole('heading', { level: 2 }).focus();
        for (let step = 0; step < 100 && !await panel.isVisible(); step += 1) await page.keyboard.press('ArrowRight');
      }
      await submit.click();
      await expect(page.getByRole('alert')).toContainText('Write the Other response');
      await expect(cards.nth(0).getByLabel('Other response')).toBeFocused();
      if (stepped) await expect(page.locator('#position')).toHaveText(/^Question 1 of 4(?:, part \d+ of \d+)?$/);
      await cards.nth(0).getByLabel('Other response').fill('Use a staged route.');
      if (stepped) {
        await cards.nth(0).getByRole('heading', { level: 2 }).focus();
        for (let step = 0; step < 100 && !await panel.isVisible(); step += 1) await page.keyboard.press('ArrowRight');
      }
      await expect(rows.nth(0)).toContainText('Other: Use a staged route.');
      await page.screenshot({ path: testInfo.outputPath('other-answer.png'), fullPage: true });
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
      const accessibility = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze();
      expect(accessibility.violations).toEqual([]);
    } finally {
      // Close Chromium's open connections before waiting for the fixture server to stop.
      await page.context().close();
      await server.stop(true);
    }
  });
}

test.describe('desktop form regressions', () => {
  test.use({ isMobile: false, hasTouch: false, userAgent: undefined });

  for (const { name, long } of [
    { name: 'Enter in Other opens Review without posting', long: false },
    { name: 'long questions split into measured parts without page scrolling', long: true }
  ]) {
    test(`desktop ${name}`, async ({ page, browserName }, testInfo) => {
      const question = structuredClone(spec.questions[0]);
      if (long) {
        delete question.flow;
        question.context = Array.from({ length: 6 }, (_, index) => `Consider requirement ${index + 1} alongside the existing workflow and its current delivery constraints.`);
        question.cards = Array.from({ length: 6 }, (_, index) => ({
          title: `Approach ${index + 1}`,
          text: 'Keep the current workflow available while comparing the delivery effort and ownership of this approach.'
        }));
        question.table = {
          columns: ['Approach', 'Delivery', 'Ownership'],
          rows: Array.from({ length: 8 }, (_, index) => [`Approach ${index + 1}`, 'Ready after the required checks finish', 'The existing team owns the workflow'])
        };
        question.flows = [
          { title: 'Prepare delivery', steps: ['Compare the available approaches.', 'Confirm the required checks.', 'Prepare the delivery.'] },
          { title: 'Extended checks', steps: Array.from({ length: 15 }, (_, index) => `Complete check ${index + 1} before proceeding to delivery.`) }
        ];
        question.options = Array.from({ length: 6 }, (_, index) => ({
          label: `Approach ${index + 1}`,
          description: 'Use this approach after confirming its delivery requirements and the responsibility for maintaining the workflow.'
        }));
      }
      const fixture = { title: 'Choose the next move', questions: long ? [question, structuredClone(spec.questions[1])] : [question] };
      const specPath = join(renderDirectory, `${long ? 'long' : 'enter'}-desktop.json`);
      await Bun.write(specPath, JSON.stringify(fixture));
      const html = await $`${command} render ${specPath}`.text();
      const postedSubmissions = [];
      let latestSubmission;
      const server = Bun.serve({
        hostname: 'localhost',
        port: 0,
        async fetch(request) {
          const pathname = new URL(request.url).pathname;
          if (pathname === '/') {
            return new Response(html, { headers: { 'Content-Type': 'text/html' } });
          } else if (pathname.startsWith('/assets/')) {
            const assetName = (await $`basename ${pathname}`.text()).trimEnd();
            return new Response(Bun.file(join(process.env.PHONE_PAGE_ASSETS_DIR, assetName)), {
              headers: { 'Content-Type': assetName.endsWith('.woff2') ? 'font/woff2' : 'text/javascript' }
            });
          } else if (pathname === '/answers' && request.method === 'POST') {
            latestSubmission = await request.json();
            postedSubmissions.push(latestSubmission);
            return new Response(null, { status: 204 });
          } else if (pathname === '/answers' && latestSubmission) {
            return Response.json(latestSubmission);
          }
          return new Response(null, { status: 404 });
        }
      });
      try {
        console.log(`Open the desktop form at 1440 by 900: ${name}.`);
        await page.setViewportSize({ width: 1440, height: 900 });
        await page.goto(`http://localhost:${server.port}/`);
        const card = page.locator('[data-question-id="route"]');
        const panel = page.locator('#answer-panel');
        const submit = page.getByRole('button', { name: 'Submit answers' });
        await expect(card).toBeVisible();
        await page.screenshot({ path: testInfo.outputPath('desktop-first-screen.png'), fullPage: true });
        if (!long) {
          // The installed WebKit build has no native textarea resize handle, including on plain HTML.
          if (browserName === 'chromium') {
            console.log('Drag the desktop note resize handle down 150 pixels.');
            const note = card.getByLabel('Note (optional)');
            for (let step = 0; step < 100 && !await note.isVisible(); step += 1) await page.locator('#next').click();
            const noteBox = await note.boundingBox();
            expect(noteBox).not.toBeNull();
            await page.mouse.move(noteBox.x + noteBox.width - 8, noteBox.y + noteBox.height - 8);
            await page.mouse.down();
            await page.mouse.move(noteBox.x + noteBox.width - 8, noteBox.y + noteBox.height + 142, { steps: 30 });
            await page.mouse.up();
            expect(await note.evaluate((element) => element.getBoundingClientRect().height) - noteBox.height).toBeGreaterThanOrEqual(100);
            await page.screenshot({ path: testInfo.outputPath('note-resized.png'), fullPage: true });
          }
          console.log('Find the answer block, select Other, and press Enter in its text field.');
          const other = card.getByLabel('Other', { exact: true });
          for (let step = 0; step < 100 && !await other.isVisible(); step += 1) await page.locator('#next').click();
          await other.check();
          const answerPosition = await page.locator('#position').textContent();
          console.log('Keep radio and modified arrow keys within the answer screen.');
          await other.focus();
          await page.keyboard.press('ArrowLeft');
          await expect(page.locator('#position')).toHaveText(answerPosition);
          await page.keyboard.press('ArrowRight');
          await expect(page.locator('#position')).toHaveText(answerPosition);
          await card.getByRole('heading', { level: 2 }).focus();
          for (const modifier of ['Alt', 'Control', 'Meta', 'Shift']) {
            await page.keyboard.press(`${modifier}+ArrowLeft`);
            await expect(page.locator('#position')).toHaveText(answerPosition);
            await page.keyboard.press(`${modifier}+ArrowRight`);
            await expect(page.locator('#position')).toHaveText(answerPosition);
          }
          await other.check();
          await card.getByLabel('Other response').fill('Use a staged route.');
          await expect(submit).toBeHidden();
          await card.getByLabel('Other response').press('Enter');
          await expect(page.getByRole('heading', { name: 'Check your answers' })).toBeFocused();
          await expect(page.locator('#position')).toHaveText('Review');
          await expect(panel).toContainText('Other: Use a staged route.');
          expect(postedSubmissions).toEqual([]);
          await page.screenshot({ path: testInfo.outputPath('enter-opens-review.png'), fullPage: true });
          console.log('Submit from Review and verify the submission time remains visible there.');
          await submit.click();
          await expect(page.getByRole('status')).toHaveText('Answers sent');
          expect(postedSubmissions).toHaveLength(1);
          expect(postedSubmissions[0].answers[0].other).toBe('Use a staged route.');
          const submittedAt = await page.evaluate((timestamp) => new Date(timestamp).toLocaleString(), postedSubmissions[0].submittedAt);
          await expect(page.locator('#reply')).toHaveText(`Sent at ${submittedAt}. Reply done in the chat`);
          await page.screenshot({ path: testInfo.outputPath('review-submitted.png'), fullPage: true });
          console.log('Reload and verify the latest submission is shown on the opening screen.');
          await page.reload();
          await expect(page.locator('.masthead #latest')).toBeVisible();
          await expect(page.locator('.masthead #latest')).toContainText('Latest submission:');
          await page.screenshot({ path: testInfo.outputPath('latest-submission.png'), fullPage: true });
        } else {
          console.log('Walk all measured parts and verify each screen fits between the fixed bars.');
          const position = page.locator('#position');
          await expect(position).toHaveText(/^Question 1 of 2, part 1 of \d+$/);
          const initialLabel = await position.textContent();
          const match = initialLabel.match(/^Question 1 of 2, part 1 of (\d+)$/);
          expect(match).not.toBeNull();
          const partCount = Number(match[1]);
          expect(partCount).toBeGreaterThan(1);
          console.log('Keep focus and scroll unchanged when Left is pressed on the first screen.');
          const initialHeading = card.locator('.question-part:not([hidden])').getByRole('heading', { level: 2 });
          await initialHeading.focus();
          await page.keyboard.press('ArrowLeft');
          await expect(initialHeading).toBeFocused();
          await expect(position).toHaveText(initialLabel);
          expect(await page.evaluate(() => window.scrollY)).toBe(0);
          const visited = [];
          const seenBlocks = [];
          let answered = false;
          let internallyScrolled = false;
          for (let part = 1; part <= partCount; part += 1) {
            await expect(position).toHaveText(`Question 1 of 2, part ${part} of ${partCount}`);
            await expect(card).toBeVisible();
            await expect(submit).toBeHidden();
            const current = card.locator('.question-part:not([hidden])');
            await expect(current).toHaveCount(1);
            await expect(current).toBeVisible();
            const fit = await current.evaluate((element) => {
              const top = document.querySelector('.progress').getBoundingClientRect().bottom;
              const bottom = document.querySelector('.navigation').getBoundingClientRect().top;
              const box = element.getBoundingClientRect();
              const blocks = [...element.querySelectorAll('.screen-column > *')].map((block) => {
                const rect = block.getBoundingClientRect();
                const range = document.createRange();
                range.selectNodeContents(block);
                const text = range.getBoundingClientRect();
                return {
                  text: block.textContent,
                  top: rect.top,
                  bottom: rect.bottom,
                  textTop: text.top,
                  textBottom: text.bottom,
                  scrolls: block.classList.contains('oversized-block'),
                  overflow: getComputedStyle(block).overflowY
                };
              });
              return { top, bottom, screenTop: box.top, screenBottom: box.bottom, blocks, scrollHeight: document.documentElement.scrollHeight, viewport: window.innerHeight, scrollY: window.scrollY };
            });
            expect(fit.screenTop).toBeGreaterThanOrEqual(fit.top - 1);
            expect(fit.screenBottom).toBeLessThanOrEqual(fit.bottom + 1);
            expect(fit.scrollHeight).toBeLessThanOrEqual(fit.viewport + 1);
            expect(fit.scrollY).toBe(0);
            for (const block of fit.blocks) {
              expect(block.top).toBeGreaterThanOrEqual(fit.top - 1);
              expect(block.bottom).toBeLessThanOrEqual(fit.bottom + 1);
              if (block.scrolls) expect(['auto', 'scroll']).toContain(block.overflow);
              else {
                expect(block.textTop).toBeGreaterThanOrEqual(fit.top - 1);
                expect(block.textBottom).toBeLessThanOrEqual(fit.bottom + 1);
              }
              seenBlocks.push(block.text);
            }
            const table = current.locator('.table-wrap');
            if (await table.count()) {
              console.log('Keep arrow keys inside the focusable table.');
              await table.focus();
              for (const key of ['ArrowLeft', 'ArrowRight']) {
                await page.keyboard.press(key);
                await expect(position).toHaveText(`Question 1 of 2, part ${part} of ${partCount}`);
                await expect(table).toBeFocused();
              }
            }
            const oversized = current.locator('.oversized-block');
            if (await oversized.count()) {
              expect(await current.locator('.screen-column > *').count()).toBe(1);
              expect(await oversized.evaluate((element) => element.scrollHeight - element.clientHeight)).toBeGreaterThan(1);
              console.log(`Scroll inside the oversized block on part ${part} and keep the document stationary.`);
              await oversized.hover();
              await page.mouse.wheel(0, 240);
              await expect.poll(() => oversized.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
              expect(await page.evaluate(() => window.scrollY)).toBe(0);
              internallyScrolled = true;
            } else {
              await page.mouse.move(2, 450);
              await page.mouse.wheel(0, 240);
              expect(await page.evaluate(() => window.scrollY)).toBe(0);
            }
            const option = card.getByLabel('Approach 1', { exact: true });
            if (await option.isVisible()) {
              await option.check();
              answered = true;
            }
            visited.push(await position.textContent());
            await page.screenshot({ path: testInfo.outputPath(`long-part-${part}.png`), fullPage: true });
            await page.locator('#next').click();
          }
          expect(visited).toHaveLength(partCount);
          expect(answered).toBe(true);
          expect(internallyScrolled).toBe(true);
          expect(seenBlocks).toHaveLength(7 + await card.locator('.context').count());
          expect(new Set(seenBlocks).size).toBe(seenBlocks.length);
          const content = seenBlocks.join('\n');
          for (const text of [question.question, ...question.context, ...question.cards.map(({ title }) => title), ...question.table.rows.flat(), ...question.flows.flatMap(({ steps }) => steps), ...question.options.map(({ label }) => label)]) expect(content).toContain(text);
          await expect(position).toHaveText(/^Question 2 of 2(?:, part 1 of \d+)?$/);
          console.log('Resize while answering the second question and keep that question active.');
          await page.locator('[data-question-id="delivery"]').getByLabel('Note (optional)').fill('Keep this question active.');
          const note = page.locator('[data-question-id="delivery"]').getByLabel('Note (optional)');
          await page.setViewportSize({ width: 420, height: 900 });
          await note.focus();
          await expect(note).toBeFocused();
          await page.setViewportSize({ width: 1280, height: 900 });
          await expect(note).toBeFocused();
          await expect(note).toHaveValue('Keep this question active.');
          await expect(position).toHaveText(/^Question 2 of 2(?:, part \d+ of \d+)?$/);
          for (let step = 0; step < 100 && !await panel.isVisible(); step += 1) await page.locator('#next').click();
          await expect(panel).toBeVisible();
          await expect(panel.locator('.answer-row').first()).toContainText('Approach 1');
          const reviewHeading = page.getByRole('heading', { name: 'Check your answers' });
          await reviewHeading.focus();
          await page.keyboard.press('ArrowRight');
          await expect(reviewHeading).toBeFocused();
          await expect(position).toHaveText('Review');
          await page.screenshot({ path: testInfo.outputPath('long-review.png'), fullPage: true });
        }
      } finally {
        await server.stop(true);
      }
    });
  }
});

test.describe('desktop list packing', () => {
  test.use({ isMobile: false, hasTouch: false });

  for (const { name, width, count, tall, tallIndex = 2 } of [
    { name: 'seven bullets use both columns', width: 1440, count: 7 },
    { name: 'bullets split at the desktop breakpoint', width: 960, count: 7 },
    { name: 'bullets continue onto another screen', width: 1440, count: 18 },
    { name: 'a tall bullet owns a scrolling screen', width: 1440, count: 7, tall: true },
    { name: 'an oversized first bullet leaves later bullets splittable', width: 1440, count: 18, tall: true, tallIndex: 0 },
    { name: 'an empty right column centers the content', width: 1440, count: 1 }
  ]) {
    test(name, async ({ page }, testInfo) => {
      const question = structuredClone(spec.questions[0]);
      delete question.flow;
      question.context = Array.from({ length: count }, (_, index) => ({
        text: `Consider requirement ${index + 1} before delivery.`,
        items: Array.from({ length: tall && index === tallIndex ? 35 : 2 }, (_, item) => `Keep detail ${index + 1}-${item + 1} with its requirement.`)
      }));
      const specPath = join(renderDirectory, `packing-${count}-${width}-${Boolean(tall)}.json`);
      await Bun.write(specPath, JSON.stringify({ title: 'Choose the next move', questions: [question] }));
      const html = await $`${command} render ${specPath}`.text();
      const server = Bun.serve({
        hostname: 'localhost',
        port: 0,
        async fetch(request) {
          const pathname = new URL(request.url).pathname;
          if (pathname === '/') {
            return new Response(html, { headers: { 'Content-Type': 'text/html' } });
          } else if (pathname.startsWith('/assets/')) {
            const assetName = (await $`basename ${pathname}`.text()).trimEnd();
            return new Response(Bun.file(join(process.env.PHONE_PAGE_ASSETS_DIR, assetName)), {
              headers: { 'Content-Type': assetName.endsWith('.woff2') ? 'font/woff2' : 'text/javascript' }
            });
          }
          return new Response(null, { status: 404 });
        }
      });
      try {
        console.log(`Open the unchanged phone layout before checking ${name}.`);
        await page.setViewportSize({ width: 420, height: 900 });
        await page.goto(`http://localhost:${server.port}/`);
        const card = page.locator('.question-card');
        const context = card.locator('.context');
        await expect(context).toHaveCount(1);
        await expect(context.locator(':scope > li')).toHaveCount(count);
        const phoneContent = await card.locator('.question-content').innerHTML();
        await page.screenshot({ path: testInfo.outputPath('packing-phone-before.png'), fullPage: true });

        for (const pass of ['initial', 'repacked']) {
          const height = count === 1 ? 1400 : 900;
          console.log(`Pack ${name} at ${width} by ${height} (${pass}).`);
          await page.setViewportSize({ width, height });
          await expect(card.locator('.question-part').first()).toBeVisible();
          await expect(card.locator('.context > li')).toHaveText(question.context.map(({ text, items }) => text + items.join('')));
          await page.screenshot({ path: testInfo.outputPath(`packing-${pass}-desktop.png`), fullPage: true });
          const first = card.locator('.question-part').first();
          if (count === 7 && !tall) {
            await expect(first.locator('.screen-column').first().locator('.context > li')).not.toHaveCount(0);
            await expect(first.locator('.screen-column').last().locator('.context > li')).not.toHaveCount(0);
            await expect(first.locator('.context > li')).toHaveCount(count);
          }
          if (count === 18 && !tall) {
            expect(await card.locator('.question-part:has(.context)').count()).toBeGreaterThan(1);
            await expect(first.locator('.screen-column').first().locator('.context > li')).not.toHaveCount(0);
            await expect(first.locator('.screen-column').last().locator('.context > li')).not.toHaveCount(0);
          }
          const parts = await card.locator('.question-part').count();
          let sawCentered = false;
          let sawOversized = false;
          for (let index = 0; index < parts; index += 1) {
            console.log(`Inspect the bullets and column geometry on ${pass} screen ${index + 1}.`);
            const part = card.locator('.question-part').nth(index);
            await expect(part).toBeVisible();
            const layout = await part.evaluate((element) => {
              const grid = element.querySelector('.part-columns');
              const columns = [...grid.children];
              const left = columns[0].getBoundingClientRect();
              const bounds = grid.parentElement.getBoundingClientRect();
              return {
                rightEmpty: columns[1].children.length === 0,
                offset: Math.abs((left.left + left.right) / 2 - (bounds.left + bounds.right) / 2),
                blocks: [...element.querySelectorAll('.screen-column > *')].map((block) => ({
                  overflow: block.getBoundingClientRect().bottom - block.parentElement.getBoundingClientRect().bottom,
                  oversized: block.classList.contains('oversized-block')
                })),
                lists: [...element.querySelectorAll('.context')].map((list) => ({
                  font: getComputedStyle(list).fontSize,
                  indent: getComputedStyle(list).paddingLeft,
                  marker: getComputedStyle(list).listStyleType,
                  scrolls: list.scrollHeight > list.clientHeight + 1,
                  items: [...list.children].map((bullet) => ({
                    text: bullet.firstChild.textContent,
                    sub: [...bullet.querySelectorAll('ul > li')].map((item) => item.textContent),
                    indent: getComputedStyle(bullet.querySelector('ul')).paddingLeft
                  }))
                }))
              };
            });
            if (layout.rightEmpty) {
              expect(layout.offset).toBeLessThanOrEqual(1);
              sawCentered = true;
            }
            for (const block of layout.blocks) expect(block.overflow).toBeLessThanOrEqual(1);
            for (const list of layout.lists) {
              expect(list).toMatchObject({ font: '17px', indent: '22px', marker: 'disc' });
              for (const bullet of list.items) {
                expect(bullet.indent).toBe('22px');
                expect(bullet.sub).toEqual(question.context.find(({ text }) => text === bullet.text).items);
              }
              if (list.scrolls) {
                expect(tall).toBe(true);
                expect(list.items).toHaveLength(1);
                expect(list.items[0].text).toBe(question.context[tallIndex].text);
                expect(layout.blocks).toEqual([{ overflow: expect.any(Number), oversized: true }]);
                console.log('Scroll the tall bullet without scrolling the desktop document.');
                const oversized = part.locator('.oversized-block');
                await oversized.hover();
                await page.mouse.wheel(0, 240);
                await expect.poll(() => oversized.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
                sawOversized = true;
              }
            }
            expect(await page.evaluate(() => window.scrollY)).toBe(0);
            await page.screenshot({ path: testInfo.outputPath(`packing-${pass}-${index + 1}.png`), fullPage: true });
            await page.locator('#next').click();
          }
          expect(sawOversized).toBe(Boolean(tall));
          if (count === 1 || tall) expect(sawCentered).toBe(true);

          console.log('Return to the phone layout and verify the original list and nested bullets are restored.');
          await page.setViewportSize({ width: 420, height: 900 });
          await expect(card.locator('.question-part')).toHaveCount(0);
          await expect(context).toHaveCount(1);
          await expect(context.locator(':scope > li')).toHaveCount(count);
          expect(await card.locator('.question-content').innerHTML()).toBe(phoneContent);
          await expect(card.locator('.oversized-block')).toHaveCount(0);
          await page.screenshot({ path: testInfo.outputPath(`packing-phone-${pass}.png`), fullPage: true });
          console.log('Select the question from the phone summary before the next desktop packing pass.');
          await page.locator('.answer-row .change').click();
        }
      } finally {
        await page.context().close();
        await server.stop(true);
      }
    });
  }
});
