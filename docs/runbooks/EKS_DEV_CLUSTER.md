# EKS Dev Cluster Runbook

This runbook covers a private, bounded Dogtap deployment for a shared EKS dev
cluster. It is for team-level telemetry inspection and contract debugging, not
for public ingress or long-term observability storage.

## Deployment Shape

Use the Kustomize overlay in `examples/deployment/eks-dev/`:

- one `Deployment` replica with `strategy: Recreate`
- private `ClusterIP` service for HTTP, APM, OTLP gRPC, and OTLP HTTP
- `ReadWriteOnce` PVC mounted at `/data`
- SQLite storage at `/data/dogtap.db`
- bounded retention through `DOGTAP_STORAGE_MAX_EVENTS` and
  `DOGTAP_STORAGE_TTL`
- explicit copy sampling through `DOGTAP_SAMPLING_RATE`
- raw payload storage disabled with `DOGTAP_ALLOW_RAW_PAYLOADS=false`
- forwarding disabled with `DOGTAP_FORWARDING_ENABLED=false`
- `NetworkPolicy` ingress limited to pods labeled `dogtap-client=true` in
  namespaces labeled `dogtap-access=true`
- non-root user, dropped capabilities, seccomp runtime default, and read-only
  root filesystem

Pin `ghcr.io/midagedev/dogtap:latest` to a released version tag for shared
trials after the first smoke succeeds.

## Apply

```bash
kubectl apply -k examples/deployment/eks-dev
kubectl -n dogtap-dev rollout status deploy/dogtap
kubectl -n dogtap-dev get pod,svc,pvc,networkpolicy
```

If your EKS cluster does not have a default StorageClass, set
`spec.storageClassName` in `examples/deployment/eks-dev/pvc.yaml` before
applying. Most EKS clusters use a gp2 or gp3 default class.

## Local Access

Keep the service private and use port-forwarding for dashboard/API inspection:

```bash
kubectl -n dogtap-dev port-forward svc/dogtap 8080:8080
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8080/readyz
curl -fsS http://127.0.0.1:8080/metrics
```

Open `http://127.0.0.1:8080/` for the dashboard.

## In-Cluster Smoke

Create a short-lived curl pod allowed by the NetworkPolicy:

```bash
kubectl -n dogtap-dev run dogtap-smoke \
  --rm -i --restart=Never \
  --labels=dogtap-client=true \
  --image=curlimages/curl -- sh
```

Inside the shell, send representative logs and RUM payloads:

```sh
curl -fsS -X POST http://dogtap:8080/api/v2/logs \
  -H 'Content-Type: application/json' \
  -d '{"service":"api","env":"dev","message":"dogtap eks smoke log","trace_id":"trace-eks-1","span_id":"span-eks-1","route":"/smoke","http.method":"POST","http.status_code":200}'

curl -fsS -X POST http://dogtap:8080/rum \
  -H 'Content-Type: application/json' \
  -d '{"service":"web","env":"dev","session":{"id":"session-eks-1"},"usr":{"id":"user-eks-1"},"view":{"url_path":"/smoke"},"context":{"account":{"id":"acct-eks"},"workspace":{"id":"ws-eks"}}}'
```

From your workstation, verify diagnostics through the port-forward:

```bash
curl -fsS -X POST http://127.0.0.1:8080/api/diagnostics \
  -H 'Content-Type: application/json' \
  -d '{"expect":{"nonEmpty":true,"sources":["logs","rum"],"services":["api","web"],"routes":["/smoke"]}}'

curl -fsS -X POST http://127.0.0.1:8080/api/v2/logs/events/search \
  -H 'Content-Type: application/json' \
  -d '{"filter":{"query":"service:api @http.status_code:200 @http.method:POST @route:/smoke"},"page":{"limit":5}}'
```

## App Wiring

For backend traces, point existing Datadog tracer settings at the private
service during the dev-cluster trial:

```text
DD_TRACE_AGENT_URL=http://dogtap.dogtap-dev.svc.cluster.local:8126
DD_AGENT_HOST=dogtap.dogtap-dev.svc.cluster.local
DD_TRACE_AGENT_PORT=8126
DD_LOGS_INJECTION=true
```

For OpenTelemetry SDKs or collectors:

```text
OTEL_EXPORTER_OTLP_ENDPOINT=http://dogtap.dogtap-dev.svc.cluster.local:4318
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://dogtap.dogtap-dev.svc.cluster.local:4318/v1/traces
OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://dogtap.dogtap-dev.svc.cluster.local:4318/v1/logs
OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://dogtap.dogtap-dev.svc.cluster.local:4318/v1/metrics
```

For Browser RUM, do not expose the whole Dogtap service publicly. Use a
private ingress, VPN-only route, or local port-forward for the RUM proxy path
only when browser-side testing requires it.

## Rollback

```bash
kubectl delete -k examples/deployment/eks-dev
kubectl -n dogtap-dev delete pvc dogtap-data
```

Deleting the PVC removes retained SQLite telemetry. Export diagnostics archives
first if the smoke evidence must be preserved.
