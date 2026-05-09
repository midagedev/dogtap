import { createServer } from 'node:http';
import { readFile } from 'node:fs/promises';
import { createReadStream } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const port = Number(process.env.RUM_APP_PORT || '18081');
const dogtapBaseUrl = process.env.DOGTAP_BASE_URL || 'http://127.0.0.1:18080';

const rumBundlePath = join(__dirname, 'node_modules', '@datadog', 'browser-rum', 'bundle', 'datadog-rum.js');

const server = createServer(async (req, res) => {
  try {
    if (req.url.startsWith('/datadog-intake-proxy')) {
      const chunks = [];
      for await (const chunk of req) {
        chunks.push(chunk);
      }
      const body = Buffer.concat(chunks);
      const target = new URL('/datadog-intake-proxy' + new URL(req.url, `http://127.0.0.1:${port}`).search, dogtapBaseUrl);
      const response = await fetch(target, {
        method: req.method,
        headers: req.headers,
        body
      });
      res.writeHead(response.status, Object.fromEntries(response.headers.entries()));
      res.end(Buffer.from(await response.arrayBuffer()));
      return;
    }

    if (req.url === '/datadog-rum.js') {
      res.writeHead(200, { 'content-type': 'application/javascript' });
      createReadStream(rumBundlePath).pipe(res);
      return;
    }

    const html = await readFile(join(__dirname, 'index.html'));
    res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' });
    res.end(html);
  } catch (error) {
    res.writeHead(500, { 'content-type': 'text/plain; charset=utf-8' });
    res.end(String(error && error.stack ? error.stack : error));
  }
});

server.listen(port, '127.0.0.1', () => {
  console.log(`rum fixture app listening on http://127.0.0.1:${port}`);
});

