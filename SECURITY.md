# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in OpenOMS, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email: **security@openoms.org**

### What to include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 1 week
- **Fix and disclosure**: We aim to release a fix within 30 days of confirmation

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest  | Yes       |

## Security Best Practices for Self-Hosting

- Always use TLS termination (nginx, Caddy, Traefik, or ingress-nginx)
- Set strong values for `JWT_SECRET` (64+ characters) and `ENCRYPTION_KEY` (64 hex chars)
- Never expose PostgreSQL or Redis ports publicly
- Use the provided Docker images with non-root users
- Rotate secrets periodically
- Keep dependencies updated (`task lint` includes security scanning)

## Release SBOMs

Every public release build generates CycloneDX JSON SBOM artifacts for the API, dashboard, and migration images after the blocking image vulnerability scan passes. The release workflow validates each SBOM and the pushed image digest before publishing the bundled `openoms-sbom-<git-sha>` artifact.

The public artifact contains image dependency inventory and image digests only. Private vulnerability monitoring, alert routing, customer communication, and incident automation are handled outside this repository by the deployment/operations layer.

## Operational Logging and Error Reporting

- Do not log raw customer PII, authorization headers, cookies, integration credentials, tokens, or secrets
- Use request IDs, tenant IDs, order IDs, and provider IDs for correlation instead of email addresses, phone numbers, or addresses
- Treat authentication, permission, and rate-limit errors as expected operational states unless they indicate abuse or a system fault
- Report unexpected server-side and dashboard runtime errors to the configured observability backend so production incidents can be triaged without exposing customer data in logs
