import { trace } from '@opentelemetry/api';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { Resource } from '@opentelemetry/resources';
import { SimpleSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { NodeTracerProvider } from '@opentelemetry/sdk-trace-node';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';

const exporter = new OTLPTraceExporter({
  url: process.env.OTEL_EXPORTER_OTLP_TRACES_ENDPOINT || 'http://127.0.0.1:14318/v1/traces'
});

const provider = new NodeTracerProvider({
  resource: new Resource({
    [SemanticResourceAttributes.SERVICE_NAME]: 'api-service',
    [SemanticResourceAttributes.SERVICE_VERSION]: 'g1-fixture',
    [SemanticResourceAttributes.DEPLOYMENT_ENVIRONMENT]: 'local'
  })
});
provider.addSpanProcessor(new SimpleSpanProcessor(exporter));
provider.register();

const tracer = trace.getTracer('dogtap-otlp-fixture');
const span = tracer.startSpan('POST /api/cases/{caseId}/exports');
span.setAttribute('http.method', 'POST');
span.setAttribute('http.route', '/api/cases/{caseId}/exports');
span.setAttribute('case.id', 'case-123');
span.setAttribute('authorization', 'Bearer fixture-token');
span.end();

await provider.shutdown();

