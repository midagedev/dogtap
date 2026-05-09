const tracer = require('dd-trace').init({
  service: process.env.DD_SERVICE || 'api-service',
  env: process.env.DD_ENV || 'local',
  version: process.env.DD_VERSION || 'g1-fixture',
  logInjection: true,
  flushInterval: 500
});

async function main() {
  const span = tracer.startSpan('web.request', {
    service: 'api-service',
    type: 'web',
    resource: 'POST /api/cases/{caseId}/exports',
    tags: {
      'http.method': 'POST',
      'http.route': '/api/cases/{caseId}/exports',
      'http.status_code': 500,
      'case.id': 'case-123',
      'workspace.id': 'workspace-123',
      'account.id': 'account-123',
      'authorization': 'Bearer fixture-token'
    }
  });

  await new Promise(resolve => setTimeout(resolve, 20));
  span.setTag('error', true);
  span.setTag('error.message', 'fixture backend error');
  span.finish();

  await new Promise(resolve => setTimeout(resolve, 2000));
}

main().catch(error => {
  console.error(error);
  process.exitCode = 1;
});

