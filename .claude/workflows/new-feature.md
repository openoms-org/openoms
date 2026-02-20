# Workflow: New Feature

## When to Use
Adding a new module, major feature, or significant functionality.

## Steps

### 1. Plan (Rafal + main session)
- Describe requirements
- EnterPlanMode, research codebase
- Propose plan: files, endpoints, schema, pages
- Rafal approves

### 2. Backend (go-dev agent)
- Migration (if needed)
- Model → Repository → Service → Handler → Route
- `go vet ./... && go test ./...`
- Update API_CONTRACTS.md

### 3. Frontend (frontend-dev agent)
- Types in api.ts → Hook → Page → Components
- `npm run build && npm run lint`
- Playwright test if applicable

### 4. Integration (integration-dev, if applicable)
- SDK package → Provider → Handler wiring
- Test against sandbox

### 5. Review (reviewer agent)
- Quality + security checklist
- Report findings

### 6. Fix & Merge
- Address findings, final tests
- Rafal merges PR, monitors deploy

### 7. Update Context
- PROJECT_STATE.md, DECISIONS.md, SECURITY_POSTURE.md

## Parallelism
- Backend + Frontend parallel if API contract agreed upfront
- Backend first if frontend depends on new endpoints
- Max 2 agents simultaneously
