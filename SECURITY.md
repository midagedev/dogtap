# Security Policy

Dogtap handles telemetry payloads, which may contain sensitive information.

## Supported Versions

Dogtap does not have a stable release line yet. Security reports should target
the `main` branch until the first versioned release is published.

## Reporting A Vulnerability

Do not open a public issue containing secrets, customer data, production
telemetry, or exploit details.

Use GitHub's private vulnerability reporting or Security Advisories for this
repository when available. If that is not available, open a minimal public issue
that says a private security report is needed, without including sensitive
details.

## Data Handling Expectations

- Do not attach raw production telemetry to public issues or pull requests.
- Use synthetic or redacted fixtures for reproduction.
- Dogtap should not display or persist API keys.
- Production modes should keep raw payload persistence disabled unless an
  operator explicitly opts in with a bounded retention policy.
