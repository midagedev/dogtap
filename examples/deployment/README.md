# Deployment Examples

These examples are starting points for team-level Dogtap trials after local
Docker Compose adoption works.

Dogtap should stay removable from the application path:

- keep the existing Datadog Agent, Datadog intake, or OpenTelemetry Collector as
  the production fidelity path
- add Dogtap as a private sidecar or companion service for short diagnostic
  windows
- bound local retention with `DOGTAP_STORAGE_MAX_EVENTS` and
  `DOGTAP_STORAGE_TTL`
- set `DOGTAP_SAMPLING_RATE` explicitly before trialing production-like traffic
- keep `DOGTAP_ALLOW_RAW_PAYLOADS=false` outside local-only debugging
- keep `DOGTAP_FORWARDING_ENABLED=false` unless the owner has approved a
  forwarding experiment and configured secrets through the deployment platform
- expose Dogtap only on private networks, port-forwarded sessions, or internal
  load balancers
- pin a released Dogtap image tag for shared trials once you choose a version;
  `latest` is convenient for the first local smoke only

Endpoint overrides in these examples preserve application instrumentation code,
not every Datadog delivery path. Dogtap currently inspects Datadog APM payloads
but does not forward APM to Datadog. Do not replace a production Datadog Agent
or collector trace path with Dogtap unless losing that telemetry during the
trial is explicitly acceptable. For a Datadog-primary production lane, use an
OpenTelemetry Collector tee or another copy path that keeps Datadog as the
primary exporter.

## Examples

| File | Use when |
| --- | --- |
| `helm-values-sidecar.yaml` | Your app Helm chart supports sidecar fragments such as `extraContainers`, `extraEnv`, and `extraVolumes`. |
| `helm-values-companion.yaml` | You want Dogtap as a private companion service that apps can target over the cluster network. |
| `ecs-task-definition.json` | You want an ECS/Fargate task definition pattern with Dogtap as a non-essential internal inspection sidecar. |

The Helm files are values fragments, not a guaranteed chart schema. Adapt the
keys to the chart you already use and keep the Dogtap env values intact unless
the safety review changes them.

Browser RUM cannot use pod or task loopback from a user's browser. For RUM,
expose only the Dogtap HTTP proxy path through a private ingress, VPN-only
route, localhost port-forward, or equivalent internal route.
