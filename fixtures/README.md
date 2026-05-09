# Fixture Inventory

The fixtures in this directory are replay smoke fixtures unless their adjacent
metadata says otherwise.

Smoke fixtures are small, hand-curated payload examples used to exercise Dogtap
decoding, normalization, validation, and replay plumbing. They are not enough to
pass G1 Fixture Evidence because G1 requires payloads captured from real SDKs,
tracers, agents, or OpenTelemetry collectors.

G1 real evidence should be captured with the scripts under `scripts/fixtures/`
and recorded under `testdata/g1-evidence/latest/` during a local run. Accepted
real fixtures can then be promoted into this directory with metadata that sets
`fixtureClass` to `g1-real-evidence`.

