---
name: create-endpoint
description: Scaffold a new REST API endpoint with handler, service method, repository method, model, and route registration
user-invocable: true
allowed-tools: Read, Write, Edit, Glob, Grep, Bash
---

# Create REST Endpoint

Scaffold a complete API endpoint for OpenOMS. Creates all layers following established patterns.

## Arguments
$ARGUMENTS should be: `<HTTP_METHOD> /v1/<path> <description>`
Example: `POST /v1/price-lists Create a new price list`

## Steps

1. **Identify the module** from the URL path (e.g., `/v1/price-lists` → price_list)

2. **Check existing files**:
   - Handler: `apps/api-server/internal/handler/{module}_handler.go`
   - Service: `apps/api-server/internal/service/{module}_service.go`
   - Repository: `apps/api-server/internal/repository/{module}_repository.go`
   - Model: `apps/api-server/internal/model/{module}.go`
   - Router: `apps/api-server/internal/router/router.go`

3. **Create/modify each layer** following patterns in `.claude/agents/go-dev.md`:
   - **Model**: Add request/response structs with `Validate()` method
   - **Repository**: Add SQL query method using `tx pgx.Tx` parameter
   - **Service**: Add business logic with `database.WithTenant()`, audit logging
   - **Handler**: Add HTTP handler with decode → validate → call service → error switch → response
   - **Router**: Register the route with appropriate middleware (auth, permissions, rate limit)

4. **Update API contracts**: Add the new endpoint to `.claude/context/API_CONTRACTS.md`

5. **Verify**: Run `go vet ./...` in `apps/api-server/`

## Checklist
- [ ] Model struct with JSON tags and Validate() method
- [ ] Repository method with parameterized SQL
- [ ] Service method with WithTenant(), validation, audit log
- [ ] Handler with proper error handling (switch on sentinel errors)
- [ ] Route registered with correct HTTP method and middleware
- [ ] JSONB fields use `string()` + `::jsonb` cast
- [ ] API_CONTRACTS.md updated
