# Security Policy

## Supported Versions

Only the latest release and the `main` branch are actively monitored and supported with security updates.

| Version | Supported          |
| ------- | ------------------ |
| Main    | :white_check_mark: |
| < 1.0.0 | :x:                |

## Security Architecture Considerations

`go-reverse-tunnel` handles raw network traffic through multiplexed channels. When deploying this package, consider the following security surfaces:

- **Authentication**: HMAC-SHA256 challenge-response mechanism using pre-shared tokens to prevent unauthorized relaying.
- **Session Transport**: Optional TLS/mTLS encryption layer to prevent eavesdropping and Man-in-the-Middle (MitM) attacks.
- **Replay Protection**: Cryptographic nonces are used during handshakes to avoid replay attacks.
- **Dashboard & API Endpoints**: Ensure monitoring HTTP endpoints are bound strictly to private interfaces or protected via internal firewalls.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability, issue, or potential flaw within `go-reverse-tunnel`:

1. Send a private report detailing the vulnerability to the project maintainer via GitHub security advisories or direct communication channels.
2. Include the following information in your report:
   - Type of issue (e.g., buffer overflow, HMAC bypass, TLS misconfiguration).
   - Full steps to reproduce or a proof-of-concept (PoC).
   - Impacted software version and target environment (OS / Go version).
   - Potential impact of the issue.

## Response Expectations

- **Initial Response**: Within 48 hours acknowledging receipt of the issue.
- **Status Update**: Within 7 business days evaluating impact and remediation steps.
- **Patch Release**: Critical security fixes will be prioritized and published immediately upon verification.

## Responsible Disclosure

We kindly ask reporters to adhere to responsible disclosure principles:
- Give us reasonable time to fix the flaw before making any public disclosures.
- Do not attempt to access or modify data belonging to other users or servers without authorization.
