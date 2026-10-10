import { execFileSync } from 'node:child_process';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import AxeBuilder from '@axe-core/playwright';
import { expect, test } from '@playwright/test';

const origin = 'https://phone-page.example';
const pageUrl = `${origin}/p/questions`;
const testDirectory = dirname(fileURLToPath(import.meta.url));
const root = resolve(testDirectory, '../..');
const command = join(root, 'phone-page/phone-page');

const spec = {
  title: 'Choose the next move',
  questions: [
    {
      id: 'route',
      header: 'Route',
      question: 'Which route should the work take?',
      context: 'The direct route is ready, but another route can be written in full.',
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
        { label: 'Access', description: 'Check keyboard and screen-reader access.' },
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
  await writeFile(specPath, JSON.stringify(spec));
  renderedPage = execFileSync(command, ['render', specPath], { encoding: 'utf8' });
});

test.afterAll(async () => {
  await rm(renderDirectory, { recursive: true, force: true });
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
      const name = basename(url.pathname);
      const contentTypes = {
        '.js': 'text/javascript',
        '.woff2': 'font/woff2'
      };
      const extension = name.endsWith('.woff2') ? '.woff2' : '.js';
      await route.fulfill({
        status: 200,
        contentType: contentTypes[extension],
        body: await readFile(join(assetsDirectory, name))
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

  await page.goto(pageUrl);
  await expect(page.getByRole('heading', { level: 1, name: spec.title })).toBeVisible();

  const questionCards = page.locator('.question-card');
  await expect(questionCards).toHaveCount(spec.questions.length);
  for (let index = 0; index < spec.questions.length; index += 1) {
    await expect(questionCards.nth(index).getByLabel('Other', { exact: true })).toBeVisible();
    await expect(questionCards.nth(index).getByLabel('Note (optional)')).toBeVisible();
  }

  const routeQuestion = page.locator('[data-question-id="route"]');
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
  await expect(page.getByText('Reply done in the chat')).toBeVisible();
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
  await expect(page.getByText('Reply done in the chat')).toBeVisible();

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
