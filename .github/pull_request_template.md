## Summary

- 

## Verification

- [ ] `go test ./...`
- [ ] `npm --prefix web run build`
- [ ] `make shell-check`
- [ ] `make deployment-check`
- [ ] `make smoke-adoption`
- [ ] Dashboard E2E or screenshot evidence, if UI changed

## Safety And Scope

- [ ] No secrets, customer data, raw production telemetry, or private adoption material are committed.
- [ ] Production-facing behavior remains bounded and fail-open where appropriate.
- [ ] Spec, gate, or decision docs are updated if behavior changed.
