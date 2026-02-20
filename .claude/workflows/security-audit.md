# Workflow: Security Audit

## When to Use
Monthly scheduled audit, or after major feature additions.

## Steps

### 1. Scan (reviewer agent or /security-audit skill)
- Read current SECURITY_POSTURE.md
- Run automated checks (SQL injection, SSRF, missing RLS, secret logging, body limits)
- Check for new dependencies with CVEs: `go list -m -json all | jq '.Path'`

### 2. Triage (Rafal)
- Classify: CRITICAL (fix now), HIGH (this sprint), MEDIUM (backlog)
- Assign to appropriate dev agent

### 3. Fix (dev agents)
- Implement fixes per finding
- Each fix reviewed by reviewer

### 4. Update
- Reviewer moves findings between Unfixed/Fixed in SECURITY_POSTURE.md
- Record date of fix

## Schedule
- Full audit: Monthly (1st week)
- Quick scan: After every major feature merge
- Dependency check: Weekly (automated in CI via Trivy)
