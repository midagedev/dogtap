import { chromium } from 'playwright';
import { mkdir, writeFile } from 'node:fs/promises';

const artifactDir = process.env.DOGTAP_FIXTURE_ARTIFACT_DIR || 'testdata/g1-evidence/latest/rum';
const appUrl = process.env.RUM_APP_URL || 'http://127.0.0.1:18081/';

await mkdir(artifactDir, { recursive: true });

const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  const network = [];
  const diagnostics = {
    console: [],
    pageErrors: [],
    requestFailures: [],
    blockedExternalRequests: []
  };

  await page.route('**/*', async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.hostname !== '127.0.0.1' && url.hostname !== 'localhost') {
      diagnostics.blockedExternalRequests.push({
        method: request.method(),
        url: request.url()
      });
      await route.abort('blockedbyclient');
      return;
    }
    await route.continue();
  });

  page.on('console', message => {
    diagnostics.console.push({
      type: message.type(),
      text: message.text()
    });
  });
  page.on('pageerror', error => {
    diagnostics.pageErrors.push(String(error && error.stack ? error.stack : error));
  });
  page.on('requestfailed', request => {
    diagnostics.requestFailures.push({
      method: request.method(),
      url: request.url(),
      failure: request.failure()
    });
  });
  page.on('request', request => {
    if (request.url().includes('/datadog-intake-proxy')) {
      network.push({
        method: request.method(),
        url: request.url(),
        headers: request.headers()
      });
    }
  });
  await page.goto(appUrl, { waitUntil: 'networkidle' });
  await page.click('#case-button');
  await page.click('#logout-button');
  try {
    await page.waitForRequest(request => request.url().includes('/datadog-intake-proxy'), { timeout: 40000 });
  } catch {
    diagnostics.console.push({
      type: 'warning',
      text: 'Timed out waiting for a Datadog RUM proxy request after 40s'
    });
  }
  await page.waitForTimeout(1000);
  await writeFile(`${artifactDir}/browser-network.json`, JSON.stringify(network, null, 2));
  await writeFile(`${artifactDir}/browser-diagnostics.json`, JSON.stringify(diagnostics, null, 2));
} finally {
  await browser.close();
}
