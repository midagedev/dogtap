import { trace } from '@opentelemetry/api';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';
import { Resource } from '@opentelemetry/resources';
import { SimpleSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { NodeTracerProvider } from '@opentelemetry/sdk-trace-node';
import { SemanticResourceAttributes } from '@opentelemetry/semantic-conventions';

const exporter = new OTLPTraceExporter({
  url: process.env.OTEL_EXPORTER_OTLP_GRPC_ENDPOINT || 'http://127.0.0.1:14317'
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

const tracer = trace.getTracer('dogtap-otlp-grpc-fixture');
const span = tracer.startSpan('dogtap.fixture.grpc');
span.setAttribute('rpc.system', 'grpc');
span.setAttribute('case.id', 'case-123');
span.setAttribute('authorization', 'Bearer fixture-token');
span.end();

await provider.shutdown();

