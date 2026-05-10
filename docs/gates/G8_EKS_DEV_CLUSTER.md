# G8 EKS Dev Cluster

Date: 2026-05-11

## Scope

This gate covers the first dev-cluster readiness slice for Dogtap on EKS. The
goal is a private, bounded inspection target that a team can apply, smoke, and
remove without introducing a public endpoint or a production telemetry store.

## Evidence

Implemented:

- EKS dev Kustomize overlay:
  `examples/deployment/eks-dev/`
- Private `ClusterIP` service for HTTP, APM, OTLP gRPC, and OTLP HTTP.
- Single-replica `Deployment` with `Recreate` strategy and SQLite PVC.
- Non-root pod/container security context, read-only root filesystem, dropped
  capabilities, and seccomp runtime default.
- Bounded retention and safety env values:
  `DOGTAP_STORAGE_MAX_EVENTS`, `DOGTAP_STORAGE_TTL`,
  `DOGTAP_SAMPLING_RATE`, `DOGTAP_ALLOW_RAW_PAYLOADS=false`, and
  `DOGTAP_FORWARDING_ENABLED=false`.
- NetworkPolicy that admits only labeled Dogtap client pods from labeled
  namespaces.
- EKS runbook:
  `docs/runbooks/EKS_DEV_CLUSTER.md`
- Deployment syntax and safety-marker check:
  `make deployment-check`

Verification:

```bash
make deployment-check
make doc-check
git diff --check
```

The repository check validates the YAML syntax and required safety markers. A
real cluster smoke still requires an EKS context and is documented in the
runbook:

```bash
kubectl apply -k examples/deployment/eks-dev
kubectl -n dogtap-dev rollout status deploy/dogtap
kubectl -n dogtap-dev port-forward svc/dogtap 8080:8080
curl -fsS http://127.0.0.1:8080/readyz
```

## Gate Status

Passed for static dev-cluster packaging and smoke runbook readiness. Live EKS
smoke evidence should be captured by the adopting team because this public
repository does not contain cluster credentials.
