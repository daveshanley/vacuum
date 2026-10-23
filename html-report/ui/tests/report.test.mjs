import assert from 'node:assert/strict';
import { before, after, test } from 'node:test';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from 'playwright';

const root = fileURLToPath(new URL('../../../', import.meta.url));
let browser, directory, reportURL;

before(async () => {
  directory = mkdtempSync(join(tmpdir(), 'vacuum-report-ui-'));
  const spec = join(directory, 'api.yaml');
  const ruleset = join(directory, 'rules.yaml');
  const report = join(directory, 'report.html');
  writeFileSync(
    spec,
    'openapi: 3.0.3\ninfo:\n  title: Example\n  version: 1.0.0\npaths: {}\n',
  );
  // Sibling rules in one category reproduce a crowded accordion.
  const rules = Object.fromEntries(
    Array.from({ length: 12 }, (_, i) => [
      `rule-${i}`,
      {
        description: `Missing field ${i}`,
        given: '$.info',
        severity: 'warn',
        category: { id: 'information', name: 'Contract Information' },
        then: { field: `missing${i}`, function: 'defined' },
      },
    ]),
  );
  writeFileSync(ruleset, JSON.stringify({ rules }));
  execFileSync(
    'go',
    [
      'run',
      '-tags',
      'html_report_ui',
      '.',
      'html-report',
      '--no-update-check',
      '--no-banner',
      '-r',
      ruleset,
      spec,
      report,
    ],
    {
      cwd: root,
      env: { ...process.env, GOWORK: 'off' },
      stdio: 'pipe',
      timeout: 300000,
    },
  );
  reportURL = pathToFileURL(report).href;
  browser = await chromium.launch({ headless: true });
});

after(async () => {
  await browser?.close();
  if (directory) rmSync(directory, { recursive: true, force: true });
});

async function openReport(viewport) {
  const page = await browser.newPage({ viewport });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto(reportURL);
  await page.waitForFunction(
    () => document.querySelector('category-piechart')?.chart,
  );
  return { page, errors };
}

test('generated report renders scores and hydrates its chart without a timer', async () => {
  const { page, errors } = await openReport({ width: 1440, height: 900 });
  const result = await page.evaluate(() => {
    const chart = document.querySelector('category-piechart');
    return {
      labels: chart.chart.data.labels,
      warnings: chart.chart.data.datasets.find(d => d.label === 'Warnings')
        .data,
      statistics: window.statistics,
      scores: document.querySelector('.category-scores').textContent,
      canvasWidth: chart.shadowRoot.querySelector('canvas').width,
    };
  });
  assert.ok(result.canvasWidth > 0);
  const overallCells = await page
    .locator('.category-scores tbody tr')
    .first()
    .locator('td')
    .allTextContents();
  const stats = result.statistics;
  assert.equal(
    Number(overallCells[1]),
    (stats.totalErrors ?? 0) +
      (stats.totalWarnings ?? 0) +
      (stats.totalInfo ?? 0) +
      (stats.totalHints ?? 0),
  );
  assert.equal(
    await page.locator('category-rule[totalrulesviolated]').count(),
    0,
  );

  assert.ok(result.scores.includes(`${result.statistics.overallScore ?? 0}%`));
  for (const [i, category] of result.statistics.categoryStatistics.entries()) {
    assert.equal(result.labels[i], category.categoryName);
    assert.equal(result.warnings[i], category.warnings);
    assert.ok(result.scores.includes(`${category.score}%`));
  }
  // Clean reports may omit the category array due to Go's omitempty JSON tags.
  await page.evaluate(() =>
    document.querySelector('category-piechart').setChartData({}),
  );
  assert.deepEqual(
    await page.evaluate(
      () => document.querySelector('category-piechart').chart.data.labels,
    ),
    [],
  );
  assert.deepEqual(errors, []);
  await page.close();
});

for (const viewport of [
  { width: 1440, height: 900 },
  { width: 960, height: 600 },
  { width: 390, height: 500 },
]) {
  test(`sibling violations remain visible at ${viewport.width}px`, async () => {
    const { page, errors } = await openReport(viewport);
    await page.locator('rule-category-link[name="information"] a').click();
    const category = page.locator('category-report#information');
    // A fixed-height category must never leave an expanded panel with no space.
    await category.evaluate(element => {
      element.style.height = '720px';
    });
    const rules = category.locator('category-rule');
    assert.equal(await rules.count(), 12);
    const first = rules.nth(0);
    const second = rules.nth(1);
    for (const rule of [first, second, first]) {
      await rule.locator('.summary').click();
      const panel = rule.locator('.violations');
      assert.ok(await panel.isVisible());
      const bounds = await panel.boundingBox();
      assert.ok(bounds.height > 10);
      assert.ok(
        await rule.locator('category-rule-result .violation').isVisible(),
      );
    }
    assert.equal(await second.locator('.violations').isVisible(), false);
    await first.locator('.summary').click();
    assert.equal(await first.locator('.violations').isVisible(), false);
    await page.locator('rule-category-link[name="all"] a').click();
    await page.locator('rule-category-link[name="information"] a').click();
    await first.locator('.summary').click();
    assert.ok(await first.locator('.violations').isVisible());
    assert.deepEqual(errors, []);
    await page.close();
  });
}
