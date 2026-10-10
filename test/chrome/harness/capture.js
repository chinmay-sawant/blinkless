#!/usr/bin/env node
// Capture Chromium box geometry for every element in one HTML fixture.
//
// Usage:
//   node test/chrome/harness/capture.js <fixture.html> <out.json> [width] [height]
//
// Part of the CSS-REVIEW-01 comparison harness under test/chrome/harness.
// Uses the puppeteer-core install under scripts/puppeteer/node_modules and
// drives the system google-chrome. Element identity for the join is the
// CSS-path built from tag:nth-of-type(n) segments, matching pathOf() in the
// Go dumper.

'use strict';

const fs = require('fs');
const path = require('path');
const puppeteer = require(path.resolve(
  __dirname,
  '..',
  '..',
  '..',
  'scripts',
  'puppeteer',
  'node_modules',
  'puppeteer-core'
));

function normalizeText(value) {
  return (value || '').replace(/\s+/g, ' ').trim();
}

async function main() {
  const [fixture, out, widthArg, heightArg] = process.argv.slice(2);
  if (!fixture || !out) {
    console.error('usage: capture.js <fixture.html> <out.json> [width] [height]');
    process.exit(2);
  }

  const width = Number.parseInt(widthArg || '1024', 10);
  const height = Number.parseInt(heightArg || '768', 10);

  const browser = await puppeteer.launch({
    executablePath: process.env.CHROME_BIN || '/usr/bin/google-chrome',
    headless: true,
    args: [
      '--no-sandbox',
      '--disable-gpu',
      '--disable-dev-shm-usage',
      '--hide-scrollbars',
      '--force-device-scale-factor=1',
      '--allow-file-access-from-files',
    ],
  });

  try {
    const version = await browser.version();
    const page = await browser.newPage();
    await page.setViewport({ width, height, deviceScaleFactor: 1 });

    // Chrome's default "standard" font on Linux is a serif (Times New Roman
    // preference), while Blinkless defaults to its embedded Liberation Sans.
    // Normalize the browser default families so unstyled text compares the
    // same face; this isolates layout from default-font selection policy.
    const cdp = await page.createCDPSession();
    await cdp.send('Page.setFontFamilies', {
      fontFamilies: {
        standard: 'Liberation Sans',
        fixed: 'Liberation Mono',
        serif: 'Liberation Sans',
        sansSerif: 'Liberation Sans',
        cursive: 'Liberation Sans',
        fantasy: 'Liberation Sans',
        math: 'Liberation Sans',
      },
    });

    await page.goto('file://' + path.resolve(fixture), { waitUntil: 'load' });
    await page.evaluate(async () => {
      if (document.fonts && document.fonts.ready) {
        await document.fonts.ready;
      }
    });
    await page.evaluate(() => new Promise((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(resolve))));

    const payload = await page.evaluate(() => {
      function pathOf(el) {
        const parts = [];
        for (let node = el; node && node.nodeType === Node.ELEMENT_NODE;
          node = node.parentElement) {
          const name = node.tagName.toLowerCase();
          let n = 1;
          for (let sib = node.previousElementSibling; sib;
            sib = sib.previousElementSibling) {
            if (sib.tagName.toLowerCase() === name) {
              n += 1;
            }
          }
          parts.unshift(name + ':nth-of-type(' + n + ')');
        }
        return parts.join('/');
      }

      function transformInfo(el) {
        // Nearest ancestor-or-self with a non-identity transform, plus whether
        // that transform is a pure translation.
        let node = el;
        let self = null;
        let ancestor = null;
        while (node && node.nodeType === Node.ELEMENT_NODE) {
          const t = getComputedStyle(node).transform;
          if (t && t !== 'none') {
            const m = t.match(/^matrix\(([^)]+)\)$/);
            const parts = m ? m[1].split(',').map(Number) : null;
            const pureTranslate = parts
              ? parts[0] === 1 && parts[3] === 1 && parts[1] === 0 && parts[2] === 0
              : false;
            if (node === el) {
              self = { transform: t, pureTranslate };
            } else {
              ancestor = { path: pathOf(node), transform: t, pureTranslate };
            }
            break;
          }
          node = node.parentElement;
        }
        return { self, ancestor };
      }

      const elements = [];
      for (const el of document.querySelectorAll('*')) {
        const rect = el.getBoundingClientRect();
        const style = getComputedStyle(el);
        const ti = transformInfo(el);
        elements.push({
          path: pathOf(el),
          tag: el.tagName.toLowerCase(),
          id: el.id || '',
          class: el.getAttribute('class') || '',
          action: el.getAttribute('data-action') || '',
          text: (el.textContent || '').replace(/\s+/g, ' ').trim(),
          x: rect.x,
          y: rect.y,
          w: rect.width,
          h: rect.height,
          display: style.display,
          visibility: style.visibility,
          position: style.position,
          transform: style.transform,
          transformSelf: ti.self,
          transformAncestor: ti.ancestor,
        });
      }

      // Font probe: proves which face the browser used for the generic
      // families, so the report can attribute text-driven deltas. The CSS
      // reset matters: fixture rules on span/p would otherwise give the probe
      // a fixed width and every family would report that width.
      const probe = document.createElement('span');
      probe.textContent = 'Www mmm 0123 .,;';
      probe.style.cssText =
        'position:absolute;left:-10000px;top:0;display:block;width:auto;' +
        'height:auto;margin:0;padding:0;border:0;font-weight:normal;' +
        'font-style:normal;letter-spacing:normal;line-height:normal;' +
        'font-size:16px;white-space:nowrap;';
      document.body.appendChild(probe);
      const fontProbe = {};
      for (const family of ['sans-serif', 'Liberation Sans', 'DejaVu Sans', 'monospace']) {
        probe.style.fontFamily = family;
        fontProbe[family] = probe.getBoundingClientRect().width;
      }
      probe.remove();

      return {
        elementCount: elements.length,
        innerWidth: window.innerWidth,
        innerHeight: window.innerHeight,
        clientWidth: document.documentElement.clientWidth,
        clientHeight: document.documentElement.clientHeight,
        scrollWidth: document.documentElement.scrollWidth,
        scrollHeight: document.documentElement.scrollHeight,
        fontProbe,
        elements,
      };
    });

    const result = {
      fixture,
      browser: {
        version,
        userAgent: await browser.userAgent(),
      },
      viewport: { width, height, deviceScaleFactor: 1 },
      fontFamilies: {
        standard: 'Liberation Sans',
        fixed: 'Liberation Mono',
        serif: 'Liberation Sans',
        sansSerif: 'Liberation Sans',
      },
      page: payload,
    };

    fs.mkdirSync(path.dirname(path.resolve(out)), { recursive: true });
    fs.writeFileSync(out, JSON.stringify(result, null, 2) + '\n');
    console.log(
      `${path.basename(fixture)}: ${payload.elementCount} elements, ` +
      `inner=${payload.innerWidth}x${payload.innerHeight}, ` +
      `client=${payload.clientWidth}x${payload.clientHeight}, ` +
      `scroll=${payload.scrollWidth}x${payload.scrollHeight}`
    );
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error('capture: ' + error.stack);
  process.exit(1);
});
