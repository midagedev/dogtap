# GitHub Actions Workflow Contract Example

This example shows the intended pattern for another repository:

1. Start Dogtap as a service container.
2. Run the repository's normal app and E2E test workflow.
3. Assert Dogtap telemetry with `dogtap diagnose -workflow-contract`.
4. Upload Dogtap diagnostics even when the contract fails.

The checked-in workflow is a template. Copy it into the target repository and
replace the placeholder app setup and E2E commands.

Recommended contract location in the target repository:

```text
.dogtap/contracts/login.yaml
```

Keep raw diagnostics artifacts ignored or short-lived. Commit only sanitized
contracts and docs.
