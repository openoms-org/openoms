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
