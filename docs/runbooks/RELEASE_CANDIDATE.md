# Release Candidate Runbook

This runbook is for maintainers preparing a public Dogtap release. It is
intentionally conservative: release only what the gates can reproduce.

## Current Release Position

Dogtap has passed first public release-candidate evidence as a local/CI
telemetry inspector. Tagging still requires this runbook's maintainer checklist
and a clean `main` branch.

The sanitized adoption profile evidence is recorded in
`docs/gates/G8_SANITIZED_ADOPTION_PROFILE.md`.

## Pre-Tag Checklist

1. Confirm the working tree is clean on `main`.
2. Review `README.md`, `CHANGELOG.md`, and `docs/SUPPORT_MATRIX.md` for claims
   that exceed implemented behavior.
3. Run the full local evidence set:

   ```bash
   go test ./...
   npm --prefix web run build
   make shell-check
   make doc-check
   make contract-check
   make smoke-adoption
   make smoke-log-bridge
   make smoke-statsd-bridge
   make smoke-external-injection
   make demo-visual-check
   go run ./cmd/dogtap replay \
     -config configs/generic-local.yaml \
     -format markdown \
     fixtures/rum/login.json \
     fixtures/logs/json-log.json \
     fixtures/apm/trace.json \
     fixtures/otlp/traces.json
   ```

4. Confirm GitHub Actions is green on `main`.
5. Re-run a public-surface scan:

   ```bash
   private_scan_regex="${DOGTAP_PRIVATE_SCAN_REGEX:-company-name|internal-host|customer-id}"
   git grep -nE "${private_scan_regex}|api[_-]?key|secret|token|password" \
     -- ':!docs/references/datadog.md' ':!.private'
   ```

   Review every hit. Public examples may mention generic token/key words only
   when they are placeholders or safety guidance.

6. Confirm `.private/` remains ignored and contains any long-running adoption
   notes that should not be published.
7. Confirm `docs/gates/G8_SANITIZED_ADOPTION_PROFILE.md` records the latest
   realistic sanitized adoption evidence.
8. Tag only after the release-candidate state is explicit in the changelog.

## Sanitized Adoption Evidence

G8 cannot pass on private raw data. Before publishing adoption evidence:

- keep raw payloads, internal notes, and long-running local evidence under
  `.private/adoption/`
- publish only sanitized summaries, fixture names, commands, validation
  outcomes, and public-safe screenshots under `docs/gates/`
- remove company names, customer identifiers, private hosts, credentials, and
  raw production telemetry
- confirm the public-surface scan has no unexplained hits

## Tag And Publish

Version tags publish binary archives and GHCR container images:

```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

The release workflow publishes:

- GitHub Release archives for Linux, macOS, and Windows
- SHA-256 checksums
- `ghcr.io/midagedev/dogtap:vX.Y.Z`
- `ghcr.io/midagedev/dogtap:latest` for stable tags only

Prerelease tags such as `vX.Y.Z-rc.1` do not update `latest`.

## Post-Tag Checks

1. Download one release archive and run `dogtap version`.
2. Pull the GHCR image and run `dogtap version`.
3. Run the container quickstart with all public intake ports exposed:

   ```bash
   docker run --rm \
     -p 8080:8080 \
     -p 8126:8126 \
     -p 4317:4317 \
     -p 4318:4318 \
     ghcr.io/midagedev/dogtap:vX.Y.Z
   ```

4. Run `make demo-seed` against the container and inspect the dashboard.
5. Confirm the GitHub Release notes, README, and support matrix still agree.

## Rollback

If the release workflow publishes a bad tag:

1. Delete or mark the GitHub Release as a prerelease with a clear note.
2. Do not move the existing tag silently. Publish a corrective patch tag.
3. If a container image was published with a bad stable tag, publish the
   corrective patch and leave an explicit release note.
