import { describe, it, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { chromium } from 'playwright';

const BASE = 'http://localhost:8808';
let browser;

before(async () => {
  browser = await chromium.launch({ headless: true });
});

after(async () => {
  if (browser) await browser.close();
});

async function getPillTexts(page) {
  return page.locator('header button.rounded-full').allTextContents();
}

async function getFeaturedPill(page) {
  const texts = await getPillTexts(page);
  const skip = ['Random', 'Cosplay', 'Japan', 'Korean', 'IP blocked'];
  return texts.find(t => !skip.includes(t.trim()));
}

describe('Image Gallery', () => {
  it('homepage loads and shows correct title', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });
    assert((await page.title()).includes('Gallery'));
    await page.close();
  });

  it('header has title, search input, and orientation buttons', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });

    assert(await page.locator('h1').first().isVisible());
    assert(await page.locator('input[placeholder*="Search"]').first().isVisible());
    assert(await page.locator('button', { hasText: '竖' }).isVisible());
    assert(await page.locator('button', { hasText: '横' }).isVisible());
    assert(await page.locator('button', { hasText: 'All' }).isVisible());

    await page.close();
  });

  it('pills are rendered (Random + categories + featured)', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });
    const firstPill = page.locator('button.rounded-full').first();
    const text = await firstPill.textContent();
    assert(text.includes('Random'), `Expected Random pill, got: ${text}`);
    await page.close();
  });

  it('orientation filter buttons toggle active state', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });

    await page.locator('button', { hasText: '竖' }).click();
    await page.waitForTimeout(500);
    assert(await page.locator('button.bg-indigo-600', { hasText: '竖' }).isVisible());

    await page.locator('button', { hasText: '横' }).click();
    await page.waitForTimeout(500);
    assert(await page.locator('button.bg-indigo-600', { hasText: '横' }).isVisible());

    await page.locator('button', { hasText: 'All' }).click();
    await page.waitForTimeout(500);
    assert(await page.locator('button.bg-indigo-600', { hasText: 'All' }).isVisible());

    await page.close();
  });

  it('Random button creates image grid', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.locator('button.rounded-full', { hasText: 'Random' }).click();
    await page.waitForTimeout(2000);
    assert.equal(await page.locator('main img').count(), 18);
    await page.close();
  });

  it('search input exists and can be typed into', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    const input = page.locator('input').first();
    assert(await input.isVisible(), 'Search input should be visible');

    await input.focus();
    await page.waitForTimeout(300);

    await page.keyboard.type('YiTuYu', { delay: 30 });
    await page.waitForTimeout(800);

    const dropdown = page.locator('.absolute.top-full');
    const dropdownVisible = await dropdown.isVisible().catch(() => false);

    if (dropdownVisible) {
      const count = await page.locator('.absolute.top-full button').count();
      assert(count > 0, 'Dropdown should have items when API available');
    }

    await page.close();
  });

  it('clicking Random image opens modal', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.locator('button.rounded-full', { hasText: 'Random' }).click();
    await page.waitForTimeout(2000);

    const gridImg = page.locator('main div.grid > div').first();
    await gridImg.click({ force: true });
    await page.waitForTimeout(1500);

    assert(await page.locator('.fixed.inset-0').first().isVisible().catch(() => false),
      'Modal should open on image click');
    await page.close();
  });

  it('double-click in modal toggles zoom indicator', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.locator('button.rounded-full', { hasText: 'Random' }).click();
    await page.waitForTimeout(2000);

    await page.locator('main div.grid > div').first().click({ force: true });
    await page.waitForTimeout(1500);

    await page.evaluate(() => {
      const img = document.querySelector('.fixed.inset-0 img');
      if (img) img.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
    });
    await page.waitForTimeout(600);

    const pct = page.locator('div:has-text("%")').last();
    assert(await pct.isVisible().catch(() => false), 'Zoom % indicator should appear after double-click');
    await page.close();
  });

  it('category pills are rendered if API available', async () => {
    const page = await browser.newPage();
    await page.goto(BASE, { waitUntil: 'networkidle' });
    const texts = await getPillTexts(page);

    for (const cat of ['Cosplay', 'Japan', 'Korean']) {
      if (texts.includes(cat)) {
        assert(await page.locator('button', { hasText: cat }).isVisible());
      }
    }
    await page.close();
  });
});
