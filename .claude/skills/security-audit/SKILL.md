---
name: security-audit
description: Run a targeted security audit on a specific module or the entire codebase
user-invocable: true
allowed-tools: Read, Glob, Grep, Bash
context: fork
agent: Explore
---

# Security Audit

Run a targeted security audit on the OpenOMS codebase. Uses the reviewer agent's security checklist.

## Arguments
$ARGUMENTS should be a module name or "full" for entire codebase.
Examples: `orders`, `auth`, `integrations`, `full`

## Audit Scope

Read `.claude/context/SECURITY_POSTURE.md` for known findings.

### Check 1: SQL Injection
```
Grep for fmt.Sprintf patterns near SQL queries:
- Pattern: `fmt.Sprintf.*SELECT|INSERT|UPDATE|DELETE`
- Exclude: parameter index patterns like `$%d` (those are safe)
- Flag: any user input interpolated into SQL string
```

### Check 2: Missing WithTenant
```
Grep for direct pool usage outside auth:
- Pattern: `pool\.Query|pool\.QueryRow|pool\.Exec`
- Exclude: files matching `auth_service|tenant_repository`
- Flag: any direct pool query in tenant-scoped code
```

### Check 3: SSRF
```
Grep for http.Client or http.Get without noPrivateDialer:
- Pattern: `http\.Client\{|http\.Get\(|http\.Post\(`
- Check: does the surrounding code use noPrivateDialer or SafeHTTPClient?
- Flag: any external HTTP call without private IP blocking
```

### Check 4: Secret Logging
```
Grep for sensitive field names in slog calls:
- Pattern: `slog\.(Info|Warn|Error|Debug).*"(password|secret|token|key|credential)`
- Flag: any log statement that might include secrets
```

### Check 5: Missing Body Limits
```
Grep for io.ReadAll without MaxBytesReader:
- Pattern: `io\.ReadAll\(r\.Body\)`
- Check: is r.Body wrapped in http.MaxBytesReader?
- Flag: any unbounded body read in webhook/upload handlers
```

### Check 6: XSS in Frontend
```
Grep for dangerouslySetInnerHTML:
- Pattern: `dangerouslySetInnerHTML`
- Flag: any usage (should be zero)
```

### Check 7: RLS on New Tables
```
Check migration files for tables without RLS:
- Read latest migration files
- For each CREATE TABLE: verify ALTER TABLE ... ENABLE ROW LEVEL SECURITY exists
- Flag: any table missing RLS policy
```

## Output
Report findings in the format defined in `.claude/agents/reviewer.md` (Review Report Format).
Update `.claude/context/SECURITY_POSTURE.md` with any new findings.
