import { expect, test } from '@playwright/test';

const root = `${import.meta.dir}/../..`;
const command = `${root}/phone-page/phone-page`;
const spec = {
  title: 'Route delivery',
  summary: 'Use the current route for delivery.',
  sections: [
    {
      heading: 'Delivery context',
      context: [
        'Use the existing route.',
        { text: 'Keep the change small.', items: ['Preserve the current lifecycle.', 'Keep delivery checks together.'] }
      ],
      cards: [
        { title: 'Ready route', text: 'The route is ready for delivery.', icon: 'check-circle-2' },
        { title: 'Delivery check', text: 'Check the visible result before delivery.' }
      ],
      table: { columns: ['Route', 'Delivery', 'Owner'], rows: [['Current', 'Ready', 'Maintainer'], ['Separate', 'Later', 'Reviewer']] },
      flows: [
        { title: 'Delivery process', steps: ['Prepare the result.', { text: 'Are checks passing?', branches: ['Deliver the result.', 'Repair the result.'] }, 'Record the result.'] },
        { title: 'Review process', steps: ['Inspect the result.', 'Record the review.'] }
      ]
    },
    {
      heading: 'Context only',
      context: [
        'The existing route has one owner.',
        { text: 'Keep each delivery readable.', items: ['Keep the original list indentation.', 'Keep each bullet with its whole text.'] },
        'The next section describes the delivery checks.'
      ]
    },
    {
      heading: 'Cards only',
      cards: [
        { title: 'Behavior', text: 'Check the behavior a reader can see.' },
        { title: 'Layout', text: 'Check the narrow and wide layouts.' }
      ]
    },
    {
      heading: 'Table only',
      table: { columns: ['Check', 'Result'], rows: [['Readable layout', 'The section remains inside the available width.'], ['Continuous delivery', 'The page preserves the whole result for review.']] }
    },
    {
      heading: 'Flow only',
      flow: ['Prepare the result.', { text: 'Is it ready?', branches: ['Deliver it.', 'Repair it.'] }, 'Record the result.']
    }
  ]
};

let renderedPage;
let renderDirectory;
let contentPolicy;
let assetNames;

test.beforeAll(async () => {
  const temporary = Bun.spawnSync(['mktemp', '-d'], { stdout: 'pipe', stderr: 'pipe' });
  expect(temporary.exitCode, temporary.stderr.toString()).toBe(0);
  renderDirectory = temporary.stdout.toString().trim();
  const specPath = `${renderDirectory}/read.json`;
  await Bun.write(specPath, JSON.stringify(spec));
  const render = Bun.spawnSync([command, 'render', specPath], { stdout: 'pipe', stderr: 'pipe' });
  expect(render.exitCode, render.stderr.toString()).toBe(0);
  renderedPage = render.stdout.toString();
  const storeSource = await Bun.file(`${root}/cf-page-store/src/index.ts`).text();
  contentPolicy = storeSource.match(/"Content-Security-Policy": "([^"]+)"/)[1];
  expect(contentPolicy).not.toContain('unsafe-eval');
  assetNames = new Set((await Bun.file(`${root}/phone-page/assets.json`).json()).assets.map((asset) => asset.name));
});

test.afterAll(() => {
  if (!renderDirectory) return;
  const cleanup = Bun.spawnSync(['rm', '-rf', '--', renderDirectory], { stdout: 'pipe', stderr: 'pipe' });
  expect(cleanup.exitCode, cleanup.stderr.toString()).toBe(0);
});

test.describe('read pages preserve their blocks across phone and desktop layouts', () => {
  for (const width of [390, 420, 959, 960, 1440]) {
    test(`read rows at ${width} pixels`, async ({ page }, testInfo) => {
      const assetsDirectory = process.env.PHONE_PAGE_ASSETS_DIR;
      expect(assetsDirectory).toBeTruthy();
      const server = Bun.serve({
        hostname: 'localhost',
        port: 0,
        fetch(request) {
          const pathname = new URL(request.url).pathname;
          if (pathname === '/p/read') {
            return new Response(renderedPage, { headers: { 'Content-Type': 'text/html', 'Content-Security-Policy': contentPolicy } });
          }
          if (pathname === '/favicon.ico') return new Response(null, { status: 204 });
          const name = pathname.slice('/assets/'.length);
          if (pathname.startsWith('/assets/') && assetNames.has(name)) {
            return new Response(Bun.file(`${assetsDirectory}/${name}`), { headers: { 'Content-Type': name.endsWith('.woff2') ? 'font/woff2' : 'text/javascript' } });
          }
          return new Response('not found', { status: 404 });
        }
      });
      const origin = server.url.origin;
      const externalRequests = [];
      const answerRequests = [];
      const browserErrors = [];
      page.on('request', (request) => {
        const url = new URL(request.url());
        if (url.origin !== origin) externalRequests.push(url.href);
        if (url.pathname.endsWith('/answers')) answerRequests.push(url.href);
      });
      page.on('pageerror', (error) => browserErrors.push(error.message));
      page.on('console', (message) => {
        if (message.type() === 'error') browserErrors.push(message.text());
      });

      try {
        console.log(`Open the rendered read page at ${width} pixels under the store content policy.`);
        await page.setViewportSize({ width, height: 900 });
        const response = await page.goto(`${origin}/p/read`);
        expect(response.headers()['content-security-policy']).toBe(contentPolicy);
        await expect(page.getByRole('heading', { level: 1, name: spec.title })).toBeVisible();
        await expect(page.locator('body')).toHaveAttribute('data-page-type', 'read');
        await page.evaluate(() => document.fonts.ready);
        await page.screenshot({ path: testInfo.outputPath('read-page-open.png'), fullPage: true });

        console.log('Inspect section order, nested bullets, cards, table cells, and CSS flow branches.');
        const sections = page.locator('.read-section');
        await expect(sections).toHaveCount(spec.sections.length);
        await expect(sections.getByRole('heading', { level: 2 })).toHaveText(spec.sections.map((section) => section.heading));
        await expect(page.locator('#intro.summary')).toHaveText(spec.summary);
        for (let index = 0; index < spec.sections.length; index += 1) {
          const section = spec.sections[index];
          const row = sections.nth(index);
          if (section.context) {
            expect(await row.locator('.context > li').evaluateAll((bullets) => bullets.map((bullet) => bullet.firstChild.textContent))).toEqual(section.context.map((item) => typeof item === 'string' ? item : item.text));
            const nested = section.context.flatMap((item) => typeof item === 'string' ? [] : item.items);
            await expect(row.locator('.context ul li')).toHaveText(nested);
          }
          if (section.cards) {
            await expect(row.locator('.card-title')).toHaveText(section.cards.map((card) => card.title));
            await expect(row.locator('.card-text')).toHaveText(section.cards.map((card) => card.text));
          }
          if (section.table) {
            await expect(row.getByRole('columnheader')).toHaveText(section.table.columns);
            await expect(row.getByRole('cell')).toHaveText(section.table.rows.flat());
          }
          if (section.flow || section.flows) {
            const flows = section.flows ?? [{ steps: section.flow }];
            await expect(row.locator('.flow')).toHaveCount(flows.length);
            for (let flowIndex = 0; flowIndex < flows.length; flowIndex += 1) {
              const flow = flows[flowIndex];
              const chart = row.locator('.flow').nth(flowIndex);
              await expect(chart.locator('.flow-step > .flow-box')).toHaveText(flow.steps.map((step) => typeof step === 'string' ? step : step.text));
              await expect(chart.locator('.flow-branch .flow-box')).toHaveText(flow.steps.flatMap((step) => typeof step === 'string' ? [] : step.branches));
            }
          }
        }

        console.log('Verify the page has no answer controls, progress bar, or paging navigation.');
        await expect(page.getByRole('progressbar')).toHaveCount(0);
        await expect(page.locator('.progress, .navigation, #question-form, .answer-controls')).toHaveCount(0);
        await expect(page.getByRole('button', { name: /^(Previous|Next|Submit|Skip|Review)/ })).toHaveCount(0);
        await expect(page.locator('input, textarea')).toHaveCount(0);

        console.log(`Measure the ${width >= 960 ? 'desktop rows and inner scrolling container' : 'single-column phone layout'} and page width.`);
        const geometry = await page.evaluate(() => {
          const rect = (element) => {
            if (!element) return null;
            const bounds = element.getBoundingClientRect();
            return { x: bounds.x, y: bounds.y, width: bounds.width, height: bounds.height, right: bounds.right, bottom: bounds.bottom };
          };
          const main = document.querySelector('main');
          const mainStyle = getComputedStyle(main);
          return {
            main: { ...rect(main), overflow: mainStyle.overflowY, scrollHeight: main.scrollHeight, clientHeight: main.clientHeight, clientWidth: main.clientWidth, padding: parseFloat(mainStyle.paddingLeft) + parseFloat(mainStyle.paddingRight) },
            masthead: rect(document.querySelector('.masthead')),
            rows: [...document.querySelectorAll('.read-section')].map((row) => ({
              ...rect(row),
              padding: parseFloat(getComputedStyle(row).paddingLeft) + parseFloat(getComputedStyle(row).paddingRight) + 2,
              copy: rect(row.querySelector('.section-content')),
              heading: rect(row.querySelector('h2')),
              cards: rect(row.querySelector('.cards')),
              table: rect(row.querySelector('.table-wrap')),
              flows: [...row.querySelectorAll(':scope > .flow')].map(rect),
              columns: getComputedStyle(row).gridTemplateColumns.split(' ').filter(Boolean).length,
              contextOnly: row.classList.contains('context-only')
            })),
            text: [...document.querySelectorAll('.context li, .card-text, td, .flow-box')].map((element) => ({ fontSize: getComputedStyle(element).fontSize })),
            bullets: [...document.querySelectorAll('.context, .context ul')].map((element) => ({ listStyle: getComputedStyle(element).listStyleType, padding: parseFloat(getComputedStyle(element).paddingLeft) })),
            scrollWidth: document.documentElement.scrollWidth,
            clientWidth: document.documentElement.clientWidth,
            documentHeight: document.scrollingElement.scrollHeight,
            documentClientHeight: document.scrollingElement.clientHeight
          };
        });
        expect(geometry.scrollWidth).toBeLessThanOrEqual(geometry.clientWidth);
        expect(geometry.main.x + geometry.main.width / 2).toBeCloseTo(width / 2, 0);
        expect(geometry.text.every((text) => text.fontSize === '17px')).toBe(true);
        expect(geometry.bullets.every((list) => list.listStyle !== 'none' && list.padding > 0)).toBe(true);
        for (const row of geometry.rows) {
          expect(row.width).toBeCloseTo(geometry.masthead.width, 0);
          expect(row.x).toBeCloseTo(geometry.masthead.x, 0);
        }

        if (width >= 960) {
          expect(geometry.main.width).toBe(Math.min(width, 1200));
          expect(geometry.main.overflow).toBe('auto');
          expect(geometry.main.scrollHeight).toBeGreaterThan(geometry.main.clientHeight);
          expect(geometry.documentHeight).toBeLessThanOrEqual(geometry.documentClientHeight);
          expect(geometry.masthead.width).toBeCloseTo(geometry.main.clientWidth - geometry.main.padding, 0);
          for (let index = 0; index < geometry.rows.length; index += 1) {
            const row = geometry.rows[index];
            if (row.cards) {
              expect(row.cards.x).toBeGreaterThan(row.copy.right);
              expect(row.cards.y).toBeCloseTo(row.copy.y, 0);
              expect(row.cards.width).toBeCloseTo(row.copy.width, 0);
            }
            for (const wide of [row.table, ...row.flows].filter(Boolean)) {
              expect(wide.width).toBeCloseTo(row.width - row.padding, 0);
              expect(wide.y).toBeGreaterThanOrEqual(Math.max(row.copy.bottom, row.cards?.bottom ?? 0));
            }
            if (index > 0) expect(row.y).toBeGreaterThan(geometry.rows[index - 1].bottom);
          }
          const contextOnly = geometry.rows[1];
          expect(contextOnly.contextOnly).toBe(true);
          expect(contextOnly.columns).toBe(1);
          expect(contextOnly.copy.x + contextOnly.copy.width / 2).toBeCloseTo(contextOnly.x + contextOnly.width / 2, 0);
          expect(contextOnly.copy.width).toBeLessThanOrEqual(720);
        } else {
          expect(geometry.main.width).toBeLessThanOrEqual(752);
          expect(geometry.documentHeight).toBeGreaterThan(geometry.documentClientHeight);
          for (const row of geometry.rows) {
            for (const block of [row.copy, row.cards, row.table, ...row.flows].filter(Boolean)) {
              expect(block.width).toBeCloseTo(row.width - row.padding, 0);
              expect(block.x).toBeCloseTo(row.copy.x, 0);
            }
            if (row.cards) expect(row.cards.y).toBeGreaterThan(row.copy.bottom);
          }
        }

        console.log('Scroll to the final section and confirm the whole read page remains reachable.');
        await sections.last().getByRole('heading', { name: 'Flow only' }).scrollIntoViewIfNeeded();
        await expect(sections.last().getByRole('heading', { name: 'Flow only' })).toBeVisible();
        await page.screenshot({ path: testInfo.outputPath('read-page-last-section.png'), fullPage: true });
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
        expect(externalRequests).toEqual([]);
        expect(answerRequests).toEqual([]);
        expect(browserErrors).toEqual([]);
      } finally {
        await page.context().close();
        server.stop(true);
      }
    });
  }
});

test.describe('read pages load mermaid only for mermaid blocks under the store content policy', () => {
  for (const { name, mermaid, roles, error } of [
    {
      name: 'state and flow diagrams render as SVG',
      mermaid: [
        { title: 'Request states', source: 'stateDiagram-v2\n  [*] --> Draft\n  Draft --> Review: Submit\n  Review --> Draft: Reject\n  Review --> Approved: Accept\n  Approved --> [*]' },
        { title: 'Release cycle', source: 'flowchart LR\n  A[Write the change] --> B[Run the checks] --> C{Checks pass?}\n  C -->|Yes| D[Merge the change] --> E[Deploy the release]\n  C -->|No| A' }
      ],
      roles: ['stateDiagram', 'flowchart-v2']
    },
    {
      name: 'an invalid diagram shows its render error',
      mermaid: [{ title: 'Broken diagram', source: 'flowchart LR\n  A -->' }],
      error: 'Parse error on line'
    },
    {
      name: 'a page without diagrams does not load mermaid'
    }
  ]) {
    test(name, async ({ page }, testInfo) => {
      const assetsDirectory = process.env.PHONE_PAGE_ASSETS_DIR;
      expect(assetsDirectory).toBeTruthy();
      const diagramSpec = {
        title: 'Request lifecycle',
        summary: 'Review the request lifecycle.',
        sections: [{ heading: 'Lifecycle', context: ['A request moves through review.'], ...(mermaid && { mermaid }) }]
      };
      const specPath = `${renderDirectory}/${testInfo.project.name}-${name.replaceAll(' ', '-')}.json`;
      await Bun.write(specPath, JSON.stringify(diagramSpec));
      const render = Bun.spawnSync([command, 'render', specPath], { stdout: 'pipe', stderr: 'pipe' });
      expect(render.exitCode, render.stderr.toString()).toBe(0);
      const html = render.stdout.toString();
      const server = Bun.serve({
        hostname: 'localhost',
        port: 0,
        fetch(request) {
          const pathname = new URL(request.url).pathname;
          if (pathname === '/p/read') {
            return new Response(html, { headers: { 'Content-Type': 'text/html', 'Content-Security-Policy': contentPolicy } });
          }
          if (pathname === '/favicon.ico') return new Response(null, { status: 204 });
          const assetName = pathname.slice('/assets/'.length);
          if (pathname.startsWith('/assets/') && assetNames.has(assetName)) {
            return new Response(Bun.file(`${assetsDirectory}/${assetName}`), { headers: { 'Content-Type': assetName.endsWith('.woff2') ? 'font/woff2' : 'text/javascript' } });
          }
          return new Response('not found', { status: 404 });
        }
      });
      const origin = server.url.origin;
      const externalRequests = [];
      const mermaidRequests = [];
      const browserErrors = [];
      page.on('request', (request) => {
        const url = new URL(request.url());
        if (url.origin !== origin) externalRequests.push(url.href);
        if (url.pathname === '/assets/mermaid.min.js') mermaidRequests.push(url.href);
      });
      page.on('pageerror', (pageError) => browserErrors.push(pageError.message));
      page.on('console', (message) => {
        if (message.type() === 'error') browserErrors.push(message.text());
      });

      try {
        const response = await page.goto(`${origin}/p/read`);
        expect(response.headers()['content-security-policy']).toBe(contentPolicy);
        await expect(page.locator('.read-section')).toHaveCount(1);
        const diagrams = page.locator('.diagram');
        await expect(diagrams.locator('.flow-label')).toHaveText((mermaid ?? []).map(({ title }) => title));
        if (roles) {
          const images = diagrams.locator('.diagram-box > svg');
          await expect(images).toHaveCount(mermaid.length);
          for (const [index, role] of roles.entries()) await expect(images.nth(index)).toHaveAttribute('aria-roledescription', role);
          const flowBox = diagrams.nth(1).locator('.diagram-box');
          expect(await flowBox.evaluate((box) => box.scrollWidth > box.clientWidth)).toBe(true);
        }
        if (error) {
          await expect(diagrams.locator('.diagram-box')).toContainText(error);
          await expect(diagrams.locator('svg')).toHaveCount(0);
        }
        await page.evaluate(() => document.fonts.ready);
        await page.screenshot({ path: testInfo.outputPath('diagrams.png'), fullPage: true });
        await expect(page.locator('script[src="/assets/mermaid.min.js"]')).toHaveCount(mermaid ? 1 : 0);
        expect(mermaidRequests).toHaveLength(mermaid ? 1 : 0);
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
        expect(externalRequests).toEqual([]);
        expect(browserErrors).toEqual([]);
      } finally {
        await page.context().close();
        server.stop(true);
      }
    });
  }
});
