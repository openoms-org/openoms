# Workflow: Bug Fix

## When to Use
Production errors, panics, 500s, broken functionality.

## Steps

### 1. Diagnose (Rafal + main session)
- Check production logs: `kubectl logs -n openoms -l app.kubernetes.io/name=openoms-api --tail=100`
- Filter: `grep -v '/health' | grep -v '/metrics'`
- Identify: error message, stack trace, affected endpoint
- Trace to source file and line

### 2. Fix (appropriate dev agent)
- go-dev for backend, frontend-dev for dashboard, integration-dev for SDK/providers
- Implement fix
- Run tests locally

### 3. Quick Review (reviewer, optional for hotfix)
- For critical production issues: skip full review, deploy immediately
- For non-urgent: run reviewer before merge

### 4. Deploy
- Push to main (admin bypass for hotfix)
- Monitor CI + deploy pipeline
- Verify fix in production logs

### 5. Post-mortem (if severity >= HIGH)
- Add to DECISIONS.md if architectural lesson learned
- Update SECURITY_POSTURE.md if security-related
- Consider adding regression test
