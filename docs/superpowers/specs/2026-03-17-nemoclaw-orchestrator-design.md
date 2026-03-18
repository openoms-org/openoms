# NemoClaw Orchestrator — Complete Design Specification

**Date:** 2026-03-17
**Author:** Rafal + Claude (brainstorming)
**Status:** Draft — awaiting approval
**Sub-project:** 2 of 3 (after CI Quality Gates, before Agent Roles & Workflows)

---

## Table of Contents

1. [Goal & Scope](#1-goal--scope)
2. [Architecture Overview](#2-architecture-overview)
3. [Component Stack](#3-component-stack)
4. [Agent Roles & Model Assignment](#4-agent-roles--model-assignment)
5. [Happy Path Flow](#5-happy-path-flow)
6. [PO Decomposition Logic](#6-po-decomposition-logic)
7. [Agent Handoff Protocol](#7-agent-handoff-protocol)
8. [Context Management Strategy](#8-context-management-strategy)
9. [Error Paths — Complete Catalog](#9-error-paths--complete-catalog)
10. [CI Retry Flow — Detailed](#10-ci-retry-flow--detailed)
11. [Conflict Resolution](#11-conflict-resolution)
12. [Escalation Matrix](#12-escalation-matrix)
13. [Marian Status Reporting](#13-marian-status-reporting)
14. [NemoClaw Installation & Deployment](#14-nemoclaw-installation--deployment)
15. [OpenRouter Integration](#15-openrouter-integration)
16. [Security Model](#16-security-model)
17. [Budget & Cost Optimization](#17-budget--cost-optimization)
18. [Monitoring & Observability](#18-monitoring--observability)
19. [State Machine — Task Lifecycle](#19-state-machine--task-lifecycle)
20. [Sequence Diagrams](#20-sequence-diagrams)
21. [Configuration Reference](#21-configuration-reference)
22. [Open Questions & Risks](#22-open-questions--risks)

---

## 1. Goal & Scope

**Goal:** Build an autonomous AI development team that handles ~80% of OpenOMS development tasks without human intervention. Humans (Rafal) handle ~20% of complex/architectural work manually via Kilo Code.

**Key clarification:** NemoClaw is NOT a fork of OpenClaw. It is an **OpenClaw plugin for NVIDIA OpenShell** — a security/orchestration layer that wraps OpenClaw inside a sandboxed environment. It uses Landlock + seccomp + network namespaces for isolation. It is currently **early-stage Alpha** software (announced GTC 2026-03-06). Expect rough edges.

**In scope:**
- Orchestrator that reads Linear tickets and dispatches to specialized AI agents
- Agents that autonomously write code, create PRs, fix CI failures, and merge
- Status reporting to Telegram/WhatsApp via Marian
- Complete error handling for every possible failure mode

**Out of scope:**
- Infrastructure provisioning (handled by Terraform in enterprise repo)
- Database migrations requiring destructive operations (always human-approved)
- Secrets management (pre-existing, never created by agents)
- Customer-facing features design (always human-driven)

**Risk note:** NemoClaw is Alpha. Fallback plan: run agents as plain Docker containers with OpenClaw directly (without NemoClaw sandbox). We lose RBAC/audit/sandbox but gain stability. Decision point: evaluate NemoClaw stability after 2 weeks of testing.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         LINEAR (OPE team)                          │
│                     Source of truth for tasks                       │
│                                                                     │
│  Ticket states: Backlog → Todo → In Progress → In Review →         │
│                 Done / Blocked / Cancelled                          │
└────────────┬────────────────────────────────────────────────────────┘
             │ webhook (issue.create, issue.update)
             │ + polling fallback every 5 min
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│              NEMOCLAW ORCHESTRATOR (Hetzner CX22)                   │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────┐      │
│  │  Scheduler / Event Loop                                   │      │
│  │  - Receives webhooks from Linear                          │      │
│  │  - Polls Linear every 5 min (fallback)                    │      │
│  │  - Maintains task queue (Redis-backed)                    │      │
│  │  - Dispatches to PO agent                                 │      │
│  │  - Tracks agent state (idle/working/blocked)              │      │
│  │  - Enforces concurrency limits (1 agent per role)         │      │
│  └──────────┬───────────────────────────────────────────────┘      │
│             │                                                       │
│  ┌──────────▼───────────────────────────────────────────────┐      │
│  │  PO Agent (Nemotron 120B via OpenRouter)                  │      │
│  │  - Reads ticket description + acceptance criteria         │      │
│  │  - Classifies: backend / frontend / fullstack / devops    │      │
│  │  - Decomposes into sub-tasks if needed                    │      │
│  │  - Assigns agent by type + updates Linear labels          │      │
│  │  - Sets priority and estimates                            │      │
│  └──────────┬───────────────────────────────────────────────┘      │
│             │ dispatches to appropriate agent                       │
│  ┌──────────▼───────────────────────────────────────────────┐      │
│  │  Dev Agents (Nemotron 120B via OpenRouter)                │      │
│  │                                                            │      │
│  │  ┌─────────────┐ ┌──────────────┐ ┌────────────────┐    │      │
│  │  │ Backend Dev │ │ Frontend Dev │ │ Integration Dev│    │      │
│  │  │ (Go)        │ │ (React/Next) │ │ (SDKs)         │    │      │
│  │  └─────────────┘ └──────────────┘ └────────────────┘    │      │
│  │                                                            │      │
│  │  ┌─────────────┐                                          │      │
│  │  │ DevOps      │                                          │      │
│  │  │ (Helm/CI)   │                                          │      │
│  │  └─────────────┘                                          │      │
│  └──────────┬───────────────────────────────────────────────┘      │
│             │ creates PR on GitHub                                   │
│  ┌──────────▼───────────────────────────────────────────────┐      │
│  │  QA Agent (Nemotron 120B via OpenRouter)                  │      │
│  │  - Triggered on new PR (GitHub webhook)                   │      │
│  │  - Runs local checks before CI                            │      │
│  │  - Monitors CI status                                     │      │
│  │  - Reads failure logs, dispatches fixes back to Dev agent │      │
│  └──────────┬───────────────────────────────────────────────┘      │
│             │ CI green → triggers review                            │
│  ┌──────────▼───────────────────────────────────────────────┐      │
│  │  Security Reviewer (Opus 4.6 via OpenRouter)              │      │
│  │  - Reviews PR diff for security issues                    │      │
│  │  - Checks OWASP Top 10, SQL injection, XSS, auth bypass  │      │
│  │  - Blocks merge if security issue found                   │      │
│  │  - Approves PR if clean                                   │      │
│  └──────────┬───────────────────────────────────────────────┘      │
│             │ escalation for architectural questions                 │
│  ┌──────────▼───────────────────────────────────────────────┐      │
│  │  CEO/Architect (Opus 4.6 via OpenRouter)                  │      │
│  │  - Escalation target for complex decisions                │      │
│  │  - Design review on changes touching 5+ files             │      │
│  │  - Final approval on breaking changes                     │      │
│  └──────────────────────────────────────────────────────────┘      │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────┐      │
│  │  State Store (Redis)                                      │      │
│  │  - Agent status: idle / working / blocked / error         │      │
│  │  - Task queue with priorities                             │      │
│  │  - CI retry counters per PR                               │      │
│  │  - Context cache (repo map, recent diffs)                 │      │
│  │  - Rate limit tracking per OpenRouter key                 │      │
│  └──────────────────────────────────────────────────────────┘      │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    GITHUB (execution layer)                          │
│                                                                     │
│  bot/OPE-xxx-description branches                                   │
│  → CI pipeline (11 jobs, 9 in auto-merge gate)                       │
│  → CodeRabbit review                                                │
│  → Auto-merge gate (all green + CodeRabbit resolved)                │
└────────────┬────────────────────────────────────────────────────────┘
             │
             ▼
┌─────────────────────────────────────────────────────────────────────┐
│              MARIAN (OpenClaw on Hetzner CX22)                      │
│                                                                     │
│  - Subscribes to orchestrator events                                │
│  - Relays to Telegram / WhatsApp                                    │
│  - Daily digest + real-time alerts                                  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. Component Stack

| Component | Technology | Where |
|-----------|-----------|-------|
| Task Management | Linear (team OPE) | SaaS |
| Orchestrator | NemoClaw (NVIDIA OpenClaw plugin) | Hetzner CX22 |
| Sandbox Runtime | OpenShell (Landlock + seccomp + netns) | Hetzner CX22 |
| Agent Gateway | OpenClaw Node.js Gateway (port 18789) | Hetzner CX22 |
| Dev Agent Models | Nemotron 3 Super 120B (12B active, 1M context) | OpenRouter API |
| Review Models | Claude Opus 4.6 | OpenRouter API |
| Code Hosting | GitHub (openoms-org/openoms) | SaaS |
| CI/CD | GitHub Actions (11 jobs, 9 in auto-merge gate) | GitHub-hosted + ARC runners |
| Code Review | CodeRabbit | SaaS (free for public repo) |
| Status Relay | Marian (OpenClaw agent) | Hetzner CX22 |
| Messaging | Telegram Bot API + WhatsApp (via Marian) | SaaS |
| State Store | Redis 7 | Hetzner CX22 (Docker) |
| Monitoring | Prometheus + Grafana (existing) | Hetzner |

---

## 4. Agent Roles & Model Assignment

| Agent | Model | Cost (OpenRouter) | Context | Primary Responsibility |
|-------|-------|-------------------|---------|----------------------|
| PO | Nemotron 3 Super 120B | $0.30/$0.75 per 1M tok | 1M | Ticket decomposition, assignment, status updates |
| Backend Dev | Nemotron 3 Super 120B | $0.30/$0.75 per 1M tok | 1M | Go code: handlers, services, repos, workers, migrations |
| Frontend Dev | Nemotron 3 Super 120B | $0.30/$0.75 per 1M tok | 1M | React/Next.js: pages, components, hooks, types |
| Integration Dev | Nemotron 3 Super 120B | $0.30/$0.75 per 1M tok | 1M | Marketplace/carrier SDKs, provider interfaces |
| DevOps | Nemotron 3 Super 120B | $0.30/$0.75 per 1M tok | 1M | Helm, CI/CD, Docker (NEVER: apply, delete, secrets) |
| QA | Nemotron 3 Super 120B | $0.30/$0.75 per 1M tok | 1M | Test execution, failure analysis, CI monitoring |
| Security Reviewer | Claude Opus 4.6 | ~$15/$75 per 1M tok | 200K | PR security review, OWASP checks, merge gating |
| CEO/Architect | Claude Opus 4.6 | ~$15/$75 per 1M tok | 200K | Escalation, design review, breaking changes |

**Concurrency rule:** Max 1 active task per agent role at a time. No parallel execution within same role — prevents file conflicts.

---

## 5. Happy Path Flow

This is the complete step-by-step flow from a Linear ticket to merged code with zero issues.

### Phase 1: Ticket Intake (0-30 seconds)

```
1. Human creates ticket in Linear (team OPE)
   - Title: "Add bulk order export endpoint"
   - Description: acceptance criteria, context
   - Labels: optional (backend, frontend, etc.)
   - Priority: Urgent / High / Medium / Low

2. Linear fires webhook → Orchestrator /api/webhooks/linear
   - Event: issue.create or issue.update (status → Todo)
   - Payload: issue ID, title, description, labels, priority

3. Orchestrator validates webhook signature
   - SHA256 HMAC with LINEAR_WEBHOOK_SECRET
   - Reject if invalid → log + alert

4. Orchestrator enqueues task in Redis
   - Key: task:{linear_issue_id}
   - Priority: maps from Linear priority
   - Status: queued
   - Timestamp: now
```

### Phase 2: PO Decomposition (30 seconds - 2 minutes)

```
5. Scheduler picks next task from queue (priority order)
   - Sets agent:po status → working
   - Sets task status → decomposing

6. PO Agent receives task context:
   - Linear ticket: title + description + acceptance criteria
   - Repo map: tree-sitter index of current main branch
   - Recent changes: git log --oneline -20 on main
   - Active PRs: gh pr list --state open (to avoid conflicts)

7. PO analyzes and classifies:
   - Type: backend / frontend / fullstack / integration / devops
   - Complexity: simple (1-2 files) / medium (3-5 files) / complex (6+ files)
   - Dependencies: does this need other tickets done first?
   - Risk: does this touch auth, billing, migrations, or security?

8. PO decides decomposition:
   a) Simple task (1-2 files, single domain) → assign directly
   b) Medium task (3-5 files, single domain) → assign directly
   c) Complex task (6+ files OR cross-domain) → decompose into sub-tasks
   d) Risky task (auth/billing/migration/security) → flag for human review

9. For decomposed tasks, PO creates Linear sub-issues:
   - Parent: original ticket
   - Each sub-task has: clear description, files to touch, acceptance criteria
   - Sub-tasks ordered by dependency (DB migration first, then API, then UI)
   - Each sub-task assigned to specific agent role

10. PO updates Linear:
    - Status: In Progress
    - Assignee: bot label (e.g., "bot:backend-dev")
    - Comment: "Decomposed into N sub-tasks: [links]"
    - Labels: agent type, complexity, risk level
```

### Phase 3: Development (2-15 minutes)

```
11. Orchestrator dispatches task to assigned Dev Agent
    - Sets agent:{role} status → working
    - Sets task status → developing

12. Dev Agent receives work package:
    - Task description + acceptance criteria
    - Relevant file contents (only files it needs to touch)
    - Related test files
    - .claude/context/ files relevant to the domain:
      - API_CONTRACTS.md (for backend/integration)
      - DOMAIN_MODEL.md (for any data changes)
      - SECURITY_POSTURE.md (for auth/security)
    - .claude/rules/ applicable to the agent type:
      - go-conventions.md (for backend)
      - react-conventions.md (for frontend)
      - migration-rules.md (for DB changes)

13. Dev Agent creates branch:
    git checkout -b bot/OPE-{issue_number}-{slug}
    - Example: bot/OPE-142-bulk-order-export

14. Dev Agent writes code:
    a) Reads existing patterns in the target files
    b) Writes failing tests first (TDD where applicable)
    c) Implements the feature/fix
    d) Runs local checks:
       - go vet ./...
       - golangci-lint run (for Go)
       - npm run lint (for frontend)
       - go test ./... -run relevant_tests (for Go)
       - npm run test (for frontend)
    e) If local checks fail → fix and retry (up to 3 local retries)

15. Dev Agent commits and pushes:
    git add <specific files>
    git commit -m "feat(orders): add bulk export endpoint

    Implements bulk CSV/JSON export for orders with date range
    and status filters. Supports streaming for large datasets.

    Closes OPE-142"
    git push -u origin bot/OPE-142-bulk-order-export

16. Dev Agent creates PR:
    gh pr create \
      --title "feat(orders): add bulk export endpoint" \
      --body "## Summary\n- Add GET /v1/orders/export endpoint\n- Support CSV and JSON formats\n- Stream response for large datasets\n\n## Linear\nCloses OPE-142\n\n## Test Plan\n- [x] Unit tests for export handler\n- [x] Unit tests for CSV/JSON formatters\n- [x] Integration test for streaming"

17. Dev Agent updates Linear:
    - Status: In Review
    - Comment: "PR created: {pr_url}"
    - Link: PR URL attached to ticket
```

### Phase 4: CI Verification (3-10 minutes)

```
18. GitHub CI triggers on PR (11 jobs total, 9 required for merge):

    REQUIRED for auto-merge gate:
    - lint-strict (Go linting, zero tolerance)
    - vet (Go static analysis)
    - test (Go unit tests, requires postgres+redis)
    - security-tests (auth/middleware tests)
    - security (govulncheck + npm audit)
    - frontend (ESLint + Vitest + Next.js build)
    - migration-safety (blocks destructive DDL unless labeled)
    - docker (multi-arch image build)
    - e2e (Playwright — has continue-on-error:true, so always "passes")

    NOT in auto-merge gate:
    - auto-format (PR-only, auto-commits fixes)
    - lint (lenient mode, only-new-issues)

19. CodeRabbit reviews PR automatically:
    - Generates review comments
    - Flags potential issues

20. QA Agent monitors CI:
    - Polls: gh pr checks {pr_number} --watch
    - Waits for all required checks to complete
    - Timeout: 15 minutes (then escalate)

21. All CI checks pass → proceed to Phase 5
    Any CI check fails → go to Section 10 (CI Retry Flow)
    CodeRabbit has comments → go to Section 9.7 (CodeRabbit handling)
```

### Phase 5: Security Review (1-3 minutes)

```
22. QA Agent triggers Security Reviewer:
    - Sends: PR diff (gh pr diff {number})
    - Sends: list of changed files
    - Sends: SECURITY_POSTURE.md for context

23. Security Reviewer (Opus 4.6) analyzes:
    - SQL injection (raw queries, missing parameterization)
    - XSS (unsanitized output, dangerouslySetInnerHTML)
    - Auth bypass (missing middleware, wrong permission checks)
    - CSRF issues (missing token validation on mutations)
    - Secrets exposure (hardcoded keys, logged tokens)
    - Dependency vulnerabilities (new packages added)
    - RLS bypass (missing WithTenant, direct pool.Query)
    - Path traversal (file operations with user input)

24. Security Reviewer verdict (enforced via GitHub PR Review API, NOT just comments):
    a) APPROVED → gh pr review {number} --approve --body "Security review: ✅ No issues found"
       This creates a formal GitHub "Approved" review.
    b) ISSUES_FOUND → gh pr review {number} --request-changes --body "{findings}"
       This blocks merge via branch protection "required reviews" rule.
       → Dev Agent fixes issues → new commit → CI re-runs → re-review
    c) CRITICAL → gh pr review {number} --request-changes + add label "security:blocked"
       → Block merge + escalate to Rafal via Telegram
       (e.g., auth bypass, SQL injection, secrets exposure)

    ENFORCEMENT: Branch protection rule requires 1 approving review from the
    security-bot GitHub account. "Request Changes" review blocks merge until
    the reviewer re-approves. The auto-merge gate also checks for the
    "security:blocked" label as a secondary safeguard.
```

### Phase 6: Merge (30 seconds)

```
25. Auto-merge gate checks:
    - All 9 auto-merge gate jobs: ✅
    - CodeRabbit threads: all resolved
    - Security review: approved (via GitHub PR review, not just comment)
    - No merge conflicts with main
    - No `security:blocked` label on PR

26. Auto-merge executes:
    - Squash merge into main
    - Delete branch bot/OPE-142-bulk-order-export
    - PR title becomes commit message

27. Post-merge updates:
    - Linear: Status → Done, comment "Merged in {commit_sha}"
    - Marian: notifies Telegram "✅ OPE-142 merged: bulk order export"
    - Redis: clear task state, agent status → idle
```

### Phase 7: Completion (immediate)

```
28. Orchestrator marks task complete:
    - Redis: delete task:{linear_issue_id}
    - Agent status: idle
    - Scheduler: pick next task from queue

29. If this was a sub-task of a decomposed ticket:
    - Check if all sub-tasks done
    - If yes → mark parent ticket Done in Linear
    - If no → continue with remaining sub-tasks
```

**Total happy path time: 7-30 minutes per ticket**

---

## 6. PO Decomposition Logic

### Classification Rules

```
IF ticket mentions ONLY Go/API/backend files:
  → assign: backend-dev

IF ticket mentions ONLY React/Next.js/frontend files:
  → assign: frontend-dev

IF ticket mentions ONLY marketplace/carrier SDK:
  → assign: integration-dev

IF ticket mentions ONLY Helm/CI/Docker/infra:
  → assign: devops

IF ticket mentions multiple domains:
  → decompose into domain-specific sub-tasks

IF ticket mentions database migration:
  → sub-task 1: migration (backend-dev)
  → sub-task 2: API changes (backend-dev, depends on 1)
  → sub-task 3: UI changes (frontend-dev, depends on 2)

IF ticket is unclear or ambiguous:
  → PO asks clarifying question as Linear comment
  → Status: Blocked
  → Wait for human response
```

### Complexity Estimation

```
Simple (≤2 files changed):
  - Bug fix in single handler
  - Add field to existing model
  - Update translation string
  - Fix lint warning
  Estimated time: 5-10 min

Medium (3-5 files changed):
  - New API endpoint (handler + service + repository + test)
  - New React component with hook
  - New integration provider method
  Estimated time: 10-20 min

Complex (6+ files OR cross-domain):
  - New feature with API + UI + tests
  - Database migration + API + UI
  - Multi-marketplace integration
  MUST decompose into sub-tasks
  Estimated time: 20-60 min

Risky (any of: auth, billing, migrations, security):
  - Flag for human review before starting
  - Security Reviewer involved from the start
  - CEO/Architect review on design
  Estimated time: varies, human-dependent
```

### Sub-task Ordering

When decomposing, PO creates sub-tasks in dependency order:

```
1. Database migrations (if any) — FIRST, always
2. Model changes (Go structs, TypeScript types)
3. Repository layer (data access)
4. Service layer (business logic)
5. Handler layer (HTTP endpoints)
6. Frontend hooks (API client)
7. Frontend components (UI)
8. Tests (if not TDD'd inline)
9. Documentation updates (API_CONTRACTS.md)
```

Each sub-task explicitly states its dependencies: `"Depends on: OPE-142-1 (migration)"`

---

## 7. Agent Handoff Protocol

### Message Format

Every agent-to-agent handoff uses a structured message:

```json
{
  "from": "po",
  "to": "backend-dev",
  "task_id": "OPE-142",
  "type": "work_assignment",
  "payload": {
    "description": "Add GET /v1/orders/export endpoint...",
    "acceptance_criteria": ["supports CSV", "supports JSON", "streams large datasets"],
    "files_to_read": [
      "apps/api-server/internal/handler/order_handler.go",
      "apps/api-server/internal/service/order_service.go",
      "apps/api-server/internal/repository/order_repository.go"
    ],
    "files_to_modify": [
      "apps/api-server/internal/handler/order_handler.go",
      "apps/api-server/internal/router/router.go"
    ],
    "files_to_create": [
      "apps/api-server/internal/handler/order_export_handler.go",
      "apps/api-server/internal/handler/order_export_handler_test.go"
    ],
    "context_docs": [
      ".claude/context/API_CONTRACTS.md",
      ".claude/rules/go-conventions.md"
    ],
    "related_prs": [],
    "dependencies": [],
    "risk_flags": []
  },
  "metadata": {
    "priority": "high",
    "complexity": "medium",
    "estimated_files": 4,
    "created_at": "2026-03-17T10:00:00Z"
  }
}
```

### Handoff Sequences

```
Standard flow:
  PO → Dev Agent → QA Agent → Security Reviewer → Auto-merge

With decomposition:
  PO → [Sub-task 1: Dev Agent A → QA] → [Sub-task 2: Dev Agent B → QA] → Security Reviewer → Auto-merge

With escalation:
  PO → Dev Agent → QA Agent → BLOCKED → PO reassesses → CEO/Architect → Dev Agent (retry)

With fix cycle:
  Dev Agent → QA Agent → CI FAIL → Dev Agent (fix) → QA Agent (re-verify) → ...
```

### Handoff Rules

1. **PO → Dev:** PO provides complete work package. Dev NEVER goes back to read the Linear ticket — all context is in the handoff.
2. **Dev → QA:** Dev provides PR number. QA reads PR diff + CI status. Dev does NOT summarize its own work (avoids hallucination).
3. **QA → Security:** QA provides PR number + "all CI green" confirmation. Security reads PR diff independently.
4. **QA → Dev (fix cycle):** QA provides exact error output (CI logs, test failures). Dev gets original work package + error context.
5. **Any → CEO/Architect:** Escalation includes: task description, what was tried, why it failed, specific question.

---

## 8. Context Management Strategy

### Problem

At 85% per-step accuracy, a 10-step workflow succeeds only 20% of the time. Wasting tokens on irrelevant context reduces accuracy further. **Token efficiency is directly proportional to task success rate.**

### Repo Map (built at startup, refreshed on main push)

Using tree-sitter, build a lightweight index of the entire repository:

```json
{
  "apps/api-server/internal/handler/order_handler.go": {
    "functions": ["HandleListOrders", "HandleGetOrder", "HandleCreateOrder", "HandleUpdateOrder"],
    "types": ["OrderListParams"],
    "imports": ["service", "model", "middleware"],
    "lines": 342
  },
  "apps/api-server/internal/service/order_service.go": {
    "functions": ["ListOrders", "GetOrder", "CreateOrder", "UpdateOrder"],
    "types": ["OrderService"],
    "imports": ["repository", "model"],
    "lines": 256
  }
  // ... entire repo
}
```

**Cost:** ~2K-5K tokens for the full map. Gives agents a bird's-eye view without reading every file.

### Context Budget Per Agent Role

| Agent | Budget | Allocation |
|-------|--------|------------|
| PO | ~10K tokens | Ticket (1K) + repo map (3K) + recent PRs (2K) + active tasks (2K) + system prompt (2K) |
| Dev Agent | ~30K tokens | System prompt (3K) + conventions (3K) + file contents (15K) + tests (5K) + repo map (3K) + task (1K) |
| QA Agent | ~15K tokens | System prompt (2K) + CI logs (5K) + PR diff (5K) + repo map (3K) |
| Security Reviewer | ~20K tokens | System prompt (3K) + PR diff (10K) + security posture (3K) + conventions (4K) |
| CEO/Architect | ~25K tokens | System prompt (3K) + full context of escalation (15K) + domain model (4K) + decisions (3K) |

### Context Loading Strategy

**Lazy loading — only load what's needed:**

```
1. Agent receives task with file list (from PO or QA)
2. Agent reads ONLY the listed files
3. If agent needs additional context → reads repo map → finds relevant files
4. Reads additional files on demand
5. NEVER reads entire directories
6. NEVER reads files outside its ownership scope
```

**Diff-based context for fix cycles:**

```
When QA sends Dev back to fix:
1. Include: original file contents (already in Dev's context)
2. Include: CI error output (new, ~1-2K tokens)
3. Include: specific failing test output
4. Do NOT re-send: conventions, repo map, task description
   (Dev already has these from initial handoff)
```

**Summarization for long-running tasks:**

```
If a task exceeds 5 round-trips (fix cycles):
1. Summarize: what was tried, what failed, current state
2. Create fresh context with summary + latest file state
3. Continue from summary (avoids context window overflow)
```

### What Each Agent Reads (and Does NOT Read)

| Agent | Reads | Does NOT Read |
|-------|-------|---------------|
| PO | Linear ticket, repo map, active PRs, agent statuses | Source code, test files |
| Backend Dev | Go source in scope, Go tests, go-conventions, API contracts | Frontend code, Helm charts, CI workflow |
| Frontend Dev | React source in scope, TS tests, react-conventions, types/api.ts | Go code, migrations, Helm charts |
| Integration Dev | SDK source, provider interfaces, integration tests | Core handler/service/repo (read-only reference only) |
| DevOps | Helm values, CI workflows, Docker files, deploy scripts | Application source code |
| QA | CI logs, PR diff, test output | Source code (reads only what fails reference) |
| Security Reviewer | PR diff, security posture, conventions | Full source (only changed code) |
| CEO/Architect | Everything relevant to the escalation question | Nothing proactively — only on-demand |

---

## 9. Error Paths — Complete Catalog

### 9.1 Linear Webhook Delivery Failure

```
Trigger: Webhook times out or returns non-2xx
Detection: Linear shows failed delivery in webhook logs
Impact: Task not picked up by orchestrator

Recovery:
  1. Polling fallback (every 5 min) catches missed webhooks
  2. Orchestrator queries: Linear API → issues with status "Todo" not in Redis queue
  3. Enqueue any missing tasks
  4. Log: "Webhook miss detected, recovered via poll: OPE-{id}"

Prevention:
  - Webhook endpoint responds within 5 seconds (ack + async processing)
  - Health check monitors webhook endpoint availability
```

### 9.2 PO Decomposition Failure

```
Trigger: PO cannot understand ticket or model returns garbage
Detection: PO output fails validation (missing required fields, invalid agent assignment)
Impact: Task stuck in "decomposing" state

Recovery:
  1. Retry with reformatted prompt (add more explicit instructions) — 1 retry
  2. If retry fails → mark ticket Blocked in Linear
  3. Add comment: "PO could not decompose this ticket. Manual review needed."
  4. Notify Rafal via Telegram
  5. Agent status → idle (accept next task)

Prevention:
  - PO prompt includes 3 worked examples of decomposition
  - Output schema validation (JSON schema for work packages)
  - Ticket template in Linear enforces structured descriptions
```

### 9.3 Dev Agent — Code Generation Failure

```
Trigger: Generated code has syntax errors, won't compile
Detection: Local checks fail (go vet, go build, npm run build)
Impact: Cannot create PR

Recovery:
  1. Dev reads error output
  2. Fixes the specific error
  3. Re-runs local checks
  4. Max 3 local fix cycles
  5. If still failing after 3 → escalate to PO with error log
  6. PO may reassign to different agent or simplify task

Prevention:
  - Dev always reads existing code patterns before writing
  - Dev runs incremental builds (not full rebuild)
  - Conventions docs loaded in context
```

### 9.4 Dev Agent — Wrong Files Modified

```
Trigger: Agent modifies files outside its ownership scope
Detection: Pre-commit hook checks modified files against agent's allowed paths
Impact: Commit rejected

Recovery:
  1. Hook outputs: "ERROR: backend-dev modified apps/dashboard/... (not in scope)"
  2. Agent reverts out-of-scope changes: git checkout -- {file}
  3. If the task genuinely requires cross-domain changes → escalate to PO
  4. PO decomposes into domain-specific sub-tasks

Prevention:
  - Agent scope defined in .claude/agents/{role}.md
  - Pre-commit hook enforces scope (git hook on the orchestrator)
  - PO work package explicitly lists files_to_modify
```

### 9.5 Dev Agent — Tests Fail

```
Trigger: Unit or integration tests fail after code changes
Detection: go test output / vitest output shows failures
Impact: Cannot push (local quality gate)

Recovery:
  1. Dev reads test failure output
  2. Identifies: is this a test I need to update, or a bug in my code?
  3. Fixes code or updates test
  4. Re-runs failing test specifically (not full suite)
  5. Max 3 local fix cycles
  6. If still failing → include test output in PR description as draft PR
  7. Escalate to QA with context

Prevention:
  - Dev reads existing tests before modifying code
  - TDD approach: write test first, then implement
```

### 9.6 CI Pipeline Failure

See Section 10 for detailed CI retry flow. Summary:

```
Trigger: Any of 9 auto-merge gate CI jobs fails
Detection: QA monitors gh pr checks
Impact: Cannot merge

Recovery: Up to 5 retry cycles (see Section 10)
Escalation: After 5 retries → Blocked + Telegram notification
```

### 9.7 CodeRabbit Review Comments

```
Trigger: CodeRabbit posts review comments on PR
Detection: QA checks for unresolved CodeRabbit threads
Impact: Cannot auto-merge (gate checks unresolved threads)

Recovery:
  1. QA reads CodeRabbit comments
  2. Classifies each comment:
     a) Actionable suggestion → dispatch to Dev agent with comment text
     b) False positive / style preference → resolve with explanation
     c) Security concern → escalate to Security Reviewer
  3. Dev pushes fix commit
  4. CodeRabbit re-reviews changed files
  5. QA resolves threads that are addressed
  6. Repeat until all threads resolved

Edge case — CodeRabbit disagrees with fix:
  1. QA reads new CodeRabbit comment on the fix
  2. If it's the same concern restated → resolve with detailed explanation
  3. If it's a new valid concern → another fix cycle
  4. Max 3 CodeRabbit cycles per PR
  5. After 3 → Rafal reviews manually
```

### 9.8 Security Reviewer Finds Issues

```
Trigger: Security Reviewer identifies vulnerability in PR
Detection: Security Reviewer posts PR comment with severity
Impact: Merge blocked

Recovery by severity:

  LOW (e.g., missing input validation on non-sensitive field):
    1. Dev agent fixes in new commit
    2. Security re-reviews only the fix
    3. If clean → approved

  MEDIUM (e.g., missing rate limiting, verbose error messages):
    1. Dev agent fixes in new commit
    2. Full security re-review
    3. May require additional changes

  HIGH (e.g., SQL injection, XSS):
    1. Dev agent fixes in new commit
    2. Full security re-review
    3. CEO/Architect also reviews
    4. Both must approve

  CRITICAL (e.g., auth bypass, RLS bypass, secrets exposure):
    1. IMMEDIATE: Telegram alert to Rafal
    2. PR marked as "DO NOT MERGE"
    3. Dev agent does NOT attempt fix (too risky for autonomous agent)
    4. Rafal reviews and fixes manually
    5. Security Reviewer re-reviews Rafal's fix
```

### 9.9 Merge Conflicts

```
Trigger: PR cannot be merged due to conflicts with main
Detection: gh pr view shows "This branch has conflicts that must be resolved"
Impact: Cannot merge

Recovery by complexity:

  Simple conflict (different regions of same file):
    1. Dev agent: git fetch origin main && git rebase origin/main
    2. Resolves conflicts (takes incoming for non-overlapping, keeps own for own changes)
    3. Force push to PR branch (only case where force push is allowed on bot/ branch)
    4. CI re-runs

  Complex conflict (same lines modified):
    1. Dev agent attempts rebase
    2. If conflict is in code Dev wrote → Dev resolves
    3. If conflict is in code another agent/human wrote → STOP
    4. Escalate to PO: "Merge conflict with {conflicting_pr} on {file}:{lines}"
    5. PO decides: which change takes priority, or manual merge needed

  Semantic conflict (no git conflict but logically incompatible):
    1. CI catches this via test failures after merge
    2. QA identifies: "Tests passed on PR branch but fail after rebase on main"
    3. Escalate to CEO/Architect for resolution

Prevention:
  - PO checks active PRs before assigning tasks
  - PO avoids assigning two tasks that touch the same files
  - Short-lived PRs (target: merge within 30 min)
```

### 9.10 OpenRouter API Failure

```
Trigger: OpenRouter returns 5xx, 429, or times out
Detection: HTTP response code / timeout exception
Impact: Agent cannot generate code

Recovery:

  Rate limit (429):
    1. Read Retry-After header
    2. Wait specified time (+ jitter)
    3. Retry request
    4. Max 5 retries with exponential backoff (1s, 2s, 4s, 8s, 16s)
    5. If still 429 → queue task, wait 60 seconds
    NOTE: Free tier = 20 req/min, 50 req/day — NOT suitable for production agents.
    Always use paid tier ($0.30/$0.75 per 1M tokens).

  Server error (5xx):
    1. Retry after 5 seconds
    2. Max 3 retries
    3. If still failing → try different provider on OpenRouter
    4. If all providers failing → park task, alert Telegram
    5. Resume when API recovers (health check every 5 min)

  Timeout (>60 seconds):
    1. Cancel request
    2. Retry with shorter prompt (summarize context)
    3. If still timing out → use smaller model as fallback
    4. Last resort: Nemotron Nano 30B (faster, less capable)

  Model unavailable:
    1. OpenRouter returns model not available
    2. Switch to alternative model:
       Nemotron 120B → Nemotron 70B → Nemotron Nano 30B
    3. Log model switch for cost tracking
    4. Alert if using fallback for >1 hour

Prevention:
  - Rate limit tracking in Redis (per API key)
  - Pre-check available tokens before long generation
  - Use streaming responses to detect failures early
```

### 9.11 GitHub API Failure

```
Trigger: gh command fails (network, auth, rate limit)
Detection: Non-zero exit code from gh CLI
Impact: Cannot create branch, PR, or read CI status

Recovery:
  Auth failure (401):
    1. Token may be expired → alert Rafal
    2. Cannot self-heal (token rotation is manual)

  Rate limit (403 with rate limit headers):
    1. Wait until X-RateLimit-Reset
    2. If urgent → use secondary GitHub token (if configured)

  Network failure:
    1. Retry 3 times with 5-second intervals
    2. If still failing → alert, park task

  PR creation failure:
    1. Check if PR already exists for this branch
    2. If yes → use existing PR
    3. If no → retry after 30 seconds
    4. If still failing → alert

Prevention:
  - Use GitHub App token (higher rate limits than PAT)
  - Track API quota in Redis
  - Batch API calls where possible
```

### 9.12 Agent Stuck in Infinite Loop

```
Trigger: Agent makes same change repeatedly, or CI fails with same error 3+ times
Detection:
  - Diff comparison: if last 2 commits have identical diffs → loop detected
  - Error comparison: if last 3 CI failures have same error signature → loop
  - Token usage: if agent uses >100K tokens without producing a commit → stuck
Impact: Wasted tokens, blocked task

Recovery:
  1. Kill current agent execution
  2. Capture: what was tried, what error persists
  3. Create fresh agent instance with:
     - Summary of what was tried
     - Explicit instruction: "Do NOT try {previously_failed_approach}"
     - Alternative approach suggestion from CEO/Architect
  4. If fresh agent also loops → escalate to human
  5. Max 2 fresh starts per task

Detection heuristics:
  - Same file modified >5 times in a row → likely loop
  - Same test failing with same error >3 times → likely loop
  - Agent requesting same file >3 times → context confusion
```

### 9.13 Context Window Overflow

```
Trigger: Agent's context exceeds model limit (1M for Nemotron, 200K for Opus)
Detection: OpenRouter returns context_length_exceeded error
Impact: Agent cannot process request

Recovery:
  1. Trigger context compaction:
     - Summarize conversation so far
     - Keep: task description, current file state, last error
     - Drop: previous file versions, resolved discussions
  2. Restart agent with compacted context
  3. If task is inherently too large → decompose further
  4. Alert PO that complexity estimate was wrong

Prevention:
  - Track token usage per agent conversation
  - Warn at 70% of limit → start proactive compaction
  - Never load more than 5 files at once
  - Use diffs instead of full files for iterations
```

### 9.14 Orchestrator Crash / Restart

```
Trigger: Process crash, OOM kill, Hetzner reboot, deployment
Detection: Process monitoring (systemd watchdog)
Impact: In-flight tasks may be lost

Recovery:
  1. Orchestrator starts → reads Redis state
  2. For each task with status "developing" or "reviewing":
     - Check: does the PR exist on GitHub?
     - If PR exists → resume from QA monitoring
     - If no PR → check: does the branch exist?
     - If branch exists → resume from "push + create PR"
     - If no branch → re-dispatch task to Dev agent (start over)
  3. For each task with status "queued" or "decomposing":
     - Re-enqueue (no work lost)
  4. For agent statuses showing "working":
     - Check if the agent's API call is still in-flight
     - If not → reset to idle

State that MUST survive restart (in Redis):
  - Task queue with all metadata
  - Agent statuses
  - CI retry counters per PR
  - Linear issue ↔ PR mapping
  - Conversation summaries per active task

State that CAN be rebuilt:
  - Repo map (rebuild from git)
  - Active PR list (query GitHub)
```

### 9.15 Linear API Failure

```
Trigger: Linear API returns errors or is unreachable
Detection: HTTP error codes / timeout
Impact: Cannot read tickets or update status

Recovery:
  1. In-flight tasks continue (they already have context)
  2. New task intake paused
  3. Status updates queued in Redis
  4. When Linear recovers → flush queued updates
  5. Alert after 15 min of downtime

Prevention:
  - Cache ticket content in Redis on first read
  - Don't depend on Linear for in-flight task execution
```

### 9.16 Marian Communication Failure

```
Trigger: Marian unreachable, Telegram API down
Detection: Message send returns error
Impact: Human not notified of status/alerts

Recovery:
  1. Queue messages in Redis
  2. Retry every 5 minutes
  3. For CRITICAL alerts: also try WhatsApp, email fallback
  4. If Marian down >1 hour: log prominently, continue work without notifications

Impact assessment: LOW — work continues, just notifications delayed
```

### 9.17 Redis Failure

```
Trigger: Redis crashes or becomes unreachable
Detection: Connection refused / timeout on Redis operations
Impact: CRITICAL — orchestrator loses state

Recovery:
  1. Orchestrator enters degraded mode:
     - No new tasks dispatched
     - In-flight tasks on agents continue (they're independent)
     - All state operations fail gracefully (log + skip)
  2. Alert Rafal IMMEDIATELY
  3. On Redis recovery:
     - If Redis had AOF persistence: state restored automatically
     - If state lost: rebuild from GitHub (PR list) + Linear (ticket status)

Prevention:
  - Redis AOF persistence enabled
  - Redis data dir on persistent volume
  - Redis memory limit set (maxmemory + allkeys-lru)
  - Daily Redis RDB snapshot backup
```

### 9.18 Human Creates PR While Agent Working on Same Area

```
Trigger: Rafal (or another human) pushes a PR that touches same files as an active agent task
Detection:
  - Agent's rebase fails with conflicts
  - Or: PO notices overlapping PRs in active PR list

Recovery:
  1. If human's PR is already merged → agent rebases on new main
  2. If human's PR is still open → STOP agent, alert PO
  3. PO decides:
     a) Agent's work supersedes → continue, human closes their PR
     b) Human's work supersedes → cancel agent's task, mark Done
     c) Both needed → coordinate merge order (human first, then agent rebases)
  4. This is communicated via Telegram

Prevention:
  - PO checks active PRs (including human PRs) before assigning
  - Rafal can mark tickets as "manual" in Linear → orchestrator skips them
```

---

## 10. CI Retry Flow — Detailed

### Attempt Cycle

```
For each CI failure (max 5 attempts per PR):

  ATTEMPT 1-4:
    1. QA detects failure: gh pr checks {number} shows ❌
    2. QA reads failed job logs: gh run view {run_id} --log-failed
    3. QA classifies error type (see Error Type Matrix below)
    4. QA dispatches fix to Dev Agent with:
       - Error type
       - Exact error output (truncated to 2K tokens max)
       - Failing job name
       - Suggested fix approach
    5. Dev Agent:
       a) Reads error output
       b) Reads relevant source file
       c) Writes fix
       d) Commits: "fix: resolve {error_type} in {job_name}"
       e) Pushes
    6. CI re-triggers automatically
    7. QA waits for CI completion
    8. If passes → proceed to Security Review
    9. If fails → increment retry counter, go to next attempt

  ATTEMPT 5 (final):
    1. QA detects failure
    2. QA does NOT dispatch fix
    3. QA posts PR comment:
       "🚫 Blocked after 5 CI retries. Last error:
        Job: {job_name}
        Error: {error_summary}

        Manual intervention needed. @rafs"
    4. Linear: Status → Blocked, comment with error details
    5. Marian → Telegram: "⚠️ OPE-{id} blocked after 5 retries: {error_summary}"
    6. Agent status → idle (accept next task)
```

### Error Type Response Matrix

| Error Type | Agent Action | Retryable? | Max Retries |
|-----------|-------------|------------|-------------|
| **Lint/format** | Apply auto-fix from linter output | Yes | 4 |
| **go vet** | Read error, fix code smell | Yes | 4 |
| **Test failure (assertion)** | Read test output, fix logic | Yes | 4 |
| **Test failure (compilation)** | Fix import/type/syntax error | Yes | 4 |
| **Security (govulncheck)** | Upgrade vulnerable dependency | Yes | 2 |
| **Security (npm audit)** | Upgrade vulnerable package | Yes | 2 |
| **e2e failure** | Read Playwright report, fix | Yes | 3 |
| **CodeRabbit comment** | Address feedback, push fix | Yes | 3 |
| **Merge conflict (simple)** | `git rebase origin/main` | Yes | 2 |
| **Merge conflict (complex)** | Escalate to PO | No | 0 |
| **Flaky test** | `gh run rerun --failed` (rerun, NOT fix) | 1 rerun | 1 |
| **Infra error** (runner down) | Wait 5 min, push empty commit to retrigger | 1 retry | 1 |
| **Docker build failure** | Read Dockerfile error, fix | Yes | 3 |
| **Timeout** | Check if test is slow or stuck, optimize | Yes | 2 |
| **OOM** | Reduce test parallelism or fix memory leak | Yes | 1 |
| **Migration safety** | Add `migration:destructive` label (human must approve) | No | 0 |
| **Unknown error** | Escalate immediately | No | 0 |

### Flaky Test Detection

```
Heuristic:
  IF test passed on the SAME commit in a previous run
  AND test fails now
  AND test is in known_flaky_tests list (maintained manually)
  THEN classify as flaky

Action for flaky:
  1. gh run rerun {run_id} --failed  (re-run only failed jobs)
  2. If passes on rerun → proceed (test was flaky)
  3. If fails again → treat as real failure
  4. Max 1 rerun per PR for flaky tests
```

---

## 11. Conflict Resolution

### Between Concurrent Agent PRs

```
Scenario: Backend Dev has PR#100, Frontend Dev has PR#101, both pass CI

Case 1: No file overlap
  → Both can merge independently
  → First to merge wins, second rebases if needed

Case 2: File overlap (e.g., both touch router.go)
  → PO detects via active PR tracking
  → PO decides merge order based on dependency
  → Second PR rebases after first merges

Case 3: Semantic conflict (e.g., Backend adds field, Frontend uses old field name)
  → CI catches after second PR rebases
  → QA identifies as integration issue
  → PO creates new ticket to fix the integration
```

### Prevention: Task Assignment Rules

```
RULE 1: Never assign two tasks that modify the same file simultaneously
  - PO checks: for each file in files_to_modify, is there an active PR touching it?
  - If yes → queue the new task until active PR merges

RULE 2: Database migrations are serialized
  - Only one migration PR at a time
  - Migration PR must merge before any dependent PRs are created

RULE 3: Router/config files are coordination points
  - router.go, api.ts (types), values.yaml
  - Only one PR can modify these at a time
  - Other agents wait
```

---

## 12. Escalation Matrix

| Situation | Escalate To | Channel | SLA |
|-----------|------------|---------|-----|
| PO can't decompose ticket | Rafal | Telegram | 4 hours |
| CI fails 5 times | Rafal | Telegram | 2 hours |
| Security CRITICAL found | Rafal | Telegram (urgent) | 30 min |
| Complex merge conflict | PO → CEO/Architect | Internal | 1 hour |
| Agent stuck in loop | PO → CEO/Architect | Internal | 1 hour |
| Unknown CI error | Rafal | Telegram | 2 hours |
| OpenRouter down >1 hour | Rafal | Telegram | Informational |
| Redis down | Rafal | Telegram (urgent) | 15 min |
| Auth/billing code change | Security + CEO | Internal | Pre-merge |
| Breaking API change | CEO/Architect | Internal | Pre-merge |
| Migration with data loss risk | Rafal | Telegram (urgent) | Pre-merge |
| Budget threshold >80% | Rafal | Telegram | Daily |
| Agent modifies wrong files | PO | Internal | Immediate |

### Escalation Format (Telegram)

```
⚠️ ESCALATION — {severity}

Task: OPE-{id} — {title}
Agent: {agent_role}
Status: {current_status}

Problem:
{1-2 sentence description}

What was tried:
{bullet list of attempts}

Action needed:
{specific ask}

PR: {github_url} (if applicable)
Linear: {linear_url}
```

### Escalation Resume Flow

When Rafal responds to an escalation, the system must resume the task:

```
How escalation resumes:

1. Rafal responds via Linear (status change or comment):
   - Linear fires webhook → Orchestrator
   - Orchestrator matches task by linear_issue_id

2. Resume paths:
   a) Rafal moves ticket from Blocked → Todo:
      → Task re-enters QUEUED state
      → PO re-decomposes (may simplify based on Rafal's comment)

   b) Rafal comments with guidance (ticket stays Blocked):
      → Orchestrator reads comment
      → Dispatches to CEO/Architect for interpretation
      → CEO/Architect creates updated work package
      → Task re-enters ASSIGNED state with new instructions

   c) Rafal moves ticket to Cancelled:
      → Task removed from Redis
      → PR closed if open: gh pr close {number}
      → Branch deleted

3. During escalation wait:
   - Agent status: IDLE (free to pick up other tasks)
   - Task status: BLOCKED in Redis + Linear
   - PR status: open (draft if incomplete) or not yet created
   - No timeout — waits indefinitely for human response
```

---

## 13. Marian Status Reporting

**Note:** Marian runs OUTSIDE the NemoClaw sandbox as a separate OpenClaw instance.
It needs network access to Telegram API (`api.telegram.org`) and WhatsApp API which
are NOT in the NemoClaw sandbox allowlist. Marian communicates with the orchestrator
via internal Docker network (redis pub/sub or HTTP).

### Message Types

**1. Real-time alerts (immediate):**
```
✅ OPE-142 merged: "Add bulk order export endpoint" (7 min)
⚠️ OPE-143 blocked after 5 CI retries (lint-strict failure)
🚨 CRITICAL security issue in OPE-144 — merge blocked, review needed
❌ OPE-145 cancelled: merge conflict with OPE-144
🔄 OPE-146 reassigned from frontend-dev to backend-dev (PO reclassified)
```

**2. Daily digest (08:00 CET):**
```
📊 OpenOMS Daily — 2026-03-17

Merged: 5 PRs
  ✅ OPE-140: Fix order status transition
  ✅ OPE-141: Add shipment tracking webhook
  ✅ OPE-142: Bulk order export endpoint
  ✅ OPE-143: Update Polish translations
  ✅ OPE-144: Allegro OAuth refresh fix

In Progress: 2
  🔄 OPE-145: Customer returns dashboard (frontend-dev, 60% done)
  🔄 OPE-146: Warehouse stock sync worker (backend-dev, PR created)

Blocked: 1
  ⚠️ OPE-147: Migration needs review (waiting for Rafal)

Queue: 3 tasks waiting

Budget: $47.20 / $280.00 (16.9% of monthly budget used)
Agents: 3/7 active, 4/7 idle
```

**3. Weekly summary (Monday 09:00 CET):**
```
📈 OpenOMS Weekly — Week 12/2026

PRs merged: 23
Avg time to merge: 14 min
CI retry rate: 21.7% (5 of 23 PRs needed retries)
Security issues found: 2 (both LOW, auto-fixed)
Human escalations: 3
Budget spent: $142.50 / $280.00 (50.9%)

Top agents by output:
  1. backend-dev: 11 PRs
  2. frontend-dev: 7 PRs
  3. integration-dev: 3 PRs
  4. devops: 2 PRs

Blocked tickets: 1 (OPE-147 — waiting 3 days)
```

### Channels

| Message Type | Telegram | WhatsApp |
|-------------|----------|----------|
| Real-time: merge | ✅ | ❌ |
| Real-time: blocked | ✅ | ✅ |
| Real-time: critical | ✅ | ✅ |
| Daily digest | ✅ | ❌ |
| Weekly summary | ✅ | ❌ |

---

## 14. NemoClaw Installation & Deployment

### Prerequisites

- Hetzner CX22 (or CX32) instance with Ubuntu 24.04
- Docker + Docker Compose installed
- Git, gh CLI installed
- Network access to: OpenRouter API, GitHub, Linear API, Telegram API

### Docker Compose Setup

```yaml
# docker-compose.nemoclaw.yml
version: "3.8"

services:
  orchestrator:
    build: ./orchestrator
    restart: unless-stopped
    environment:
      - LINEAR_API_KEY=${LINEAR_API_KEY}
      - LINEAR_WEBHOOK_SECRET=${LINEAR_WEBHOOK_SECRET}
      - LINEAR_TEAM_ID=OPE
      - OPENROUTER_API_KEY=${OPENROUTER_API_KEY}
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - GITHUB_REPO=openoms-org/openoms
      - REDIS_URL=redis://redis:6379
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}
      - MARIAN_URL=http://marian:3000
      - LOG_LEVEL=info
      - MAX_CI_RETRIES=5
      - BUDGET_MONTHLY_LIMIT=280
    ports:
      - "8090:8090"  # webhook receiver
    volumes:
      - ./workspaces:/workspaces  # agent git worktrees
      - orchestrator-data:/data    # logs, state backups
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3

  marian:
    image: openclaw/openclaw:latest  # or specific version
    restart: unless-stopped
    environment:
      - OPENCLAW_CONFIG=/config/marian.yml
    volumes:
      - ./config/marian.yml:/config/marian.yml:ro
    depends_on:
      - redis

volumes:
  redis-data:
  orchestrator-data:
```

### NemoClaw Configuration

```yaml
# config/nemoclaw.yml
runtime:
  sandbox: true
  rbac:
    enabled: true
    roles:
      - name: dev-agent
        permissions:
          - git.clone
          - git.push
          - git.branch.create
          - git.branch.delete
          - file.read
          - file.write
          - shell.execute
        denied:
          - git.push.force
          - git.push.main
          - shell.sudo
          - file.write:/etc/*
          - file.write:~/.ssh/*
      - name: reviewer-agent
        permissions:
          - git.clone
          - file.read
          - github.pr.comment
          - github.pr.review
        denied:
          - file.write
          - git.push
          - shell.execute

audit:
  enabled: true
  log_path: /data/audit.log
  events:
    - agent.start
    - agent.stop
    - file.write
    - git.push
    - shell.execute
    - api.call

privacy:
  router:
    enabled: true
    rules:
      - block: secrets/*
      - block: .env*
      - block: "*.pem"
      - block: "*.key"
      - redact: DATABASE_URL
      - redact: JWT_SECRET
      - redact: ENCRYPTION_KEY

providers:
  openrouter:
    base_url: https://openrouter.ai/api/v1
    api_key: ${OPENROUTER_API_KEY}
    models:
      dev:
        id: nvidia/nemotron-3-super-120b-a12b
        max_tokens: 8192
        temperature: 0.1
      review:
        id: anthropic/claude-opus-4-6
        max_tokens: 4096
        temperature: 0.0
    fallback:
      - nvidia/nemotron-3-super-120b-a12b
      - nvidia/nemotron-3-super-70b
      - nvidia/nemotron-3-nano-30b-a3b
```

### First-time Setup

```bash
# 1. Clone repo on Hetzner
ssh openoms@hetzner-cx22
git clone https://github.com/openoms-org/openoms.git /opt/openoms
cd /opt/openoms

# 2. Install NemoClaw
curl -fsSL https://docs.nvidia.com/nemoclaw/install.sh | bash
nemoclaw setup --config config/nemoclaw.yml

# 3. Create .env file (NEVER commit)
cat > .env << 'EOF'
LINEAR_API_KEY=lin_api_xxx
LINEAR_WEBHOOK_SECRET=whsec_xxx
OPENROUTER_API_KEY=sk-or-xxx
GITHUB_TOKEN=ghp_xxx
TELEGRAM_BOT_TOKEN=123456:ABC-xxx
TELEGRAM_CHAT_ID=-100xxx
EOF

# 4. Start services
docker compose -f docker-compose.nemoclaw.yml up -d

# 5. Configure Linear webhook
# URL: https://nemoclaw.openoms.org/api/webhooks/linear
# Events: Issues (create, update)

# 6. Verify
curl http://localhost:8090/health
docker compose logs -f orchestrator
```

### Git Workspace Strategy

Each agent task gets an isolated git worktree from a shared bare repo:

```
/opt/openoms.git/               ← shared bare repo (git clone --bare)
/workspaces/
  OPE-142-bulk-export/          ← git worktree for task OPE-142
  OPE-143-translations/         ← git worktree for task OPE-143

Setup (once):
  git clone --bare https://github.com/openoms-org/openoms.git /opt/openoms.git

Per task:
  cd /opt/openoms.git
  git fetch origin main
  git worktree add /workspaces/OPE-{id}-{slug} -b bot/OPE-{id}-{slug} origin/main

Cleanup (after merge or cancel):
  git worktree remove /workspaces/OPE-{id}-{slug}
  git branch -d bot/OPE-{id}-{slug}

Disk estimate:
  Bare repo: ~200MB (shared, fetched once)
  Each worktree: ~50MB (working tree only, no .git duplication)
  5 concurrent tasks: ~450MB total

Cleanup policy:
  - On task DONE: remove worktree immediately after merge confirmed
  - On task BLOCKED: keep worktree for 24h (human may resume)
  - On task CANCELLED: remove worktree immediately
  - Daily cron: remove any orphaned worktrees older than 48h
```

### Task Deduplication

```
Deduplication is enforced at enqueue time via Redis:

  ENQUEUE(linear_issue_id):
    IF EXISTS task:{linear_issue_id} IN Redis:
      SKIP (already tracked)
    ELSE:
      SET task:{linear_issue_id} = {status: "queued", ...}

This applies to BOTH webhook-triggered and polling-triggered intake.
The polling fallback (every 5 min) queries Linear for issues with
status "Todo" and only enqueues those NOT already in Redis.
```

---

## 15. OpenRouter Integration

### API Key Management

```
- Single API key for all agents (simplifies management)
- Rate limit tracking in Redis per model
- Budget tracking per agent role (for cost attribution)
- Monthly spend cap: $280 (hard limit in orchestrator config)
```

### Request Pattern

```python
# Pseudocode for agent API call
def call_openrouter(agent_role, messages, task_id):
    model = MODEL_MAP[agent_role]  # e.g., "nvidia/nemotron-3-super-120b-a12b"

    # Check rate limit
    if redis.get(f"ratelimit:{model}:remaining") <= 0:
        wait_until = redis.get(f"ratelimit:{model}:reset")
        sleep_until(wait_until)

    # Check budget
    monthly_spend = redis.get("budget:monthly:total")
    if monthly_spend >= BUDGET_LIMIT * 0.95:
        alert_telegram("Budget at 95%! Pausing non-critical tasks.")
        if not is_critical(task_id):
            return BUDGET_EXCEEDED

    response = openrouter.chat.completions.create(
        model=model,
        messages=messages,
        max_tokens=8192,
        temperature=0.1,
        stream=True,  # streaming for early failure detection
        headers={
            "HTTP-Referer": "https://openoms.org",
            "X-Title": "OpenOMS NemoClaw",
        }
    )

    # Track usage
    usage = response.usage
    cost = calculate_cost(model, usage.prompt_tokens, usage.completion_tokens)
    redis.incrbyfloat("budget:monthly:total", cost)
    redis.incrbyfloat(f"budget:monthly:{agent_role}", cost)
    redis.set(f"ratelimit:{model}:remaining",
              response.headers["x-ratelimit-remaining"])

    return response
```

### Fallback Chain

```
Primary: nvidia/nemotron-3-super-120b-a12b ($0.30/$0.75 per 1M)
  ↓ if unavailable or rate limited
Fallback 1: nvidia/nemotron-3-super-70b (cheaper, less capable)
  ↓ if unavailable
Fallback 2: nvidia/nemotron-3-nano-30b-a3b (fast, limited capability)
  ↓ if ALL nvidia models down
Emergency: anthropic/claude-sonnet-4-6 (more expensive, high quality)

For review agents (already using Opus):
Primary: anthropic/claude-opus-4-6
  ↓ if unavailable
Fallback: anthropic/claude-sonnet-4-6
```

---

## 16. Security Model

### NEVER Rules (Hard Constraints — Cannot Be Overridden)

```
Agents MUST NEVER:
  1. Force push to main/master
  2. Run terraform apply
  3. Run kubectl delete in production namespace
  4. Create, modify, or read secrets (.env, credentials, keys)
  5. Merge without ALL CI checks green
  6. Merge without CodeRabbit threads resolved
  7. Merge with unresolved CRITICAL security finding
  8. Modify .github/workflows/* on main directly (CI pipeline changes allowed
     on bot/ branches by DevOps agent, but require human review before merge)
  9. Modify NemoClaw configuration
  10. Access other tenants' data
  11. Disable tests, linting, or security checks
  12. Add dependencies not in approved registry
  13. Execute arbitrary shell commands outside sandbox
  14. Access network resources not in allowlist
```

### NemoClaw Sandbox Enforcement

```
Network allowlist (per agent):
  - github.com (git operations)
  - api.github.com (API calls)
  - openrouter.ai (model inference)
  - api.linear.app (task management)
  - registry.npmjs.org (npm install — read only)
  - proxy.golang.org (go mod — read only)

Everything else: BLOCKED by NemoClaw sandbox

File system restrictions:
  - Write: only within /workspaces/{task_id}/
  - Read: repo files + .claude/ context files
  - Blocked: /etc/*, ~/.ssh/*, ~/.config/*, any dotenv files
```

### Audit Trail

```
Every agent action logged to /data/audit.log:
{
  "timestamp": "2026-03-17T10:15:32Z",
  "agent": "backend-dev",
  "task": "OPE-142",
  "action": "file.write",
  "target": "apps/api-server/internal/handler/order_export_handler.go",
  "bytes": 2340,
  "result": "success"
}

Retention: 90 days
Alerts: any denied action triggers immediate Telegram notification
```

---

## 17. Budget & Cost Optimization

### Monthly Budget: $280

| Category | Allocation | Notes |
|----------|-----------|-------|
| Dev agents (Nemotron 120B) | ~$120 | $0.30/$0.75 per 1M tokens, ~40 tasks/month |
| QA agent (Nemotron 120B) | ~$40 | Smaller prompts, mostly CI log analysis |
| PO agent (Nemotron 120B) | ~$20 | Short prompts, classification only |
| Security Reviewer (Opus 4.6) | ~$60 | Expensive but critical, ~40 reviews/month |
| CEO/Architect (Opus 4.6) | ~$20 | Rare escalation, ~5-10/month |
| Marian + overhead | ~$20 | OpenClaw free tier + Telegram API |

### Cost Optimization Strategies

```
1. Token efficiency:
   - Repo map instead of full file loads (~90% reduction)
   - Diffs for fix cycles instead of full files
   - Context compaction at 70% budget
   - Streaming responses for early error detection

2. Model selection:
   - Nemotron for routine work (10x cheaper than Opus)
   - Opus only for security + escalation
   - Nano fallback for simple tasks (5x cheaper than full Nemotron)

3. Task batching:
   - PO groups related small tasks into single PR
   - QA reviews multiple PRs in batch (shared CI monitoring)

4. Budget alerts:
   - 50% → info message in daily digest
   - 80% → Telegram warning
   - 95% → pause non-critical tasks
   - 100% → stop all agents, alert Rafal
```

---

## 18. Monitoring & Observability

### Metrics (Prometheus)

```
# Agent metrics
nemoclaw_agent_tasks_total{agent="backend-dev", status="completed"} counter
nemoclaw_agent_tasks_total{agent="backend-dev", status="failed"} counter
nemoclaw_agent_active{agent="backend-dev"} gauge (0 or 1)
nemoclaw_agent_task_duration_seconds{agent="backend-dev"} histogram

# CI metrics
nemoclaw_ci_retries_total{pr_number="100"} counter
nemoclaw_ci_first_pass_rate gauge (percentage)
nemoclaw_ci_duration_seconds histogram

# Cost metrics
nemoclaw_openrouter_cost_usd{agent="backend-dev", model="nemotron-120b"} counter
nemoclaw_openrouter_tokens_total{type="input"} counter
nemoclaw_openrouter_tokens_total{type="output"} counter
nemoclaw_budget_remaining_usd gauge

# Queue metrics
nemoclaw_queue_depth gauge
nemoclaw_queue_wait_seconds histogram

# Error metrics
nemoclaw_errors_total{type="api_failure"} counter
nemoclaw_errors_total{type="ci_failure"} counter
nemoclaw_errors_total{type="loop_detected"} counter
nemoclaw_escalations_total{severity="critical"} counter
```

### Grafana Dashboards

```
Dashboard 1: NemoClaw Overview
  - Tasks completed today / this week / this month
  - Agents status (idle/working/blocked)
  - CI first-pass rate trend
  - Budget usage gauge
  - Queue depth over time

Dashboard 2: Agent Performance
  - Per-agent: tasks completed, avg time, retry rate
  - Token usage per agent
  - Model fallback frequency
  - Error rate per agent

Dashboard 3: Cost Analysis
  - Daily/weekly/monthly spend
  - Cost per task (by complexity)
  - Cost per agent role
  - Projected monthly total
```

### Health Checks

```
GET /health — overall orchestrator health
{
  "status": "healthy",
  "checks": {
    "redis": "connected",
    "linear_api": "reachable",
    "openrouter_api": "reachable",
    "github_api": "reachable",
    "marian": "connected",
    "agents": {
      "po": "idle",
      "backend_dev": "working",
      "frontend_dev": "idle",
      "integration_dev": "idle",
      "devops": "idle",
      "qa": "working",
      "security_reviewer": "idle",
      "ceo_architect": "idle"
    },
    "queue_depth": 3,
    "budget_remaining_usd": 232.80
  }
}

Health check frequency: every 30 seconds
Alert if unhealthy for >5 minutes
```

---

## 19. State Machine — Task Lifecycle

```
                    ┌──────────┐
                    │  QUEUED   │ ← Linear webhook / poll
                    └─────┬────┘
                          │ PO picks up
                    ┌─────▼────────┐
                    │ DECOMPOSING  │ ← PO analyzing
                    └─────┬────────┘
                          │
              ┌───────────┼───────────┐
              │           │           │
         (simple)    (complex)   (unclear)
              │           │           │
              │     ┌─────▼────┐  ┌──▼───────┐
              │     │ SPLIT    │  │ BLOCKED   │ → waiting human
              │     │ (parent) │  │ (needs    │   clarification
              │     └─────┬────┘  │  input)   │
              │           │       └─────┬─────┘
              │     creates sub-tasks   │ human responds
              │     (each follows this  │
              │      same state machine)│
              │                         │
              │◀────────────────────────┘ (back to DECOMPOSING)
              │
        ┌─────▼────────┐
        │  ASSIGNED     │ ← dispatched to agent
        └─────┬────────┘
              │
        ┌─────┼──────────┐
        │                 │
   (agent starts)    (dispatch fail: API down, agent busy)
        │                 │
        │           ┌─────▼────────┐
        │           │  QUEUED      │ ← re-queued, retry later
        │           └──────────────┘
        │
        ┌─────▼────────┐
        │  DEVELOPING   │ ← agent writing code
        └─────┬────────┘
              │
        ┌─────┼──────────┐
        │     │           │
    (success) │     (local fail, max 3 cycles)
        │     │           │
        │     │     ┌─────▼────────┐
        │     │     │  LOCAL_FIX   │ ← fixing local errors
        │     │     └─────┬────────┘
        │     │           │
        │     │     ┌─────┼──────────┐
        │     │     │                 │
        │     │  (fixed)        (3 cycles exhausted)
        │     │     │                 │
        │     │     │◀────────┐ ┌────▼─────────┐
        │     │     └─────────┘ │  ESCALATED   │
        │     │                 └────┬─────────┘
        │     │                      │
        │     │          ┌───────────┼───────────┐
        │     │          │           │           │
        │     │     (PO simplifies) (CEO guides) (too complex)
        │     │          │           │           │
        │     │     ┌────▼───┐ ┌────▼───┐  ┌───▼─────┐
        │     │     │ASSIGNED│ │DEVELOP │  │ BLOCKED │ → human
        │     │     │(re-try)│ │-ING   │  └─────────┘
        │     │     └────────┘ └────────┘
        │     │
        ┌─────▼────────┐
        │  PR_CREATED   │ ← PR on GitHub
        └─────┬────────┘
              │
        ┌─────▼────────┐
        │  CI_RUNNING   │ ← waiting for CI
        └─────┬────────┘
              │
        ┌─────┼──────────┐
        │     │           │
    (all pass) │     (fail)
        │     │           │
        │     │     ┌─────▼────────┐
        │     │     │  CI_FIXING   │ ← fix cycle
        │     │     └─────┬────────┘
        │     │           │
        │     │     ┌─────┼──────────┐
        │     │     │                 │
        │     │  (fixed → CI_RUNNING) (5 retries exhausted)
        │     │     │                 │
        │     │     └────▶CI_RUNNING  │
        │     │                  ┌────▼────────┐
        │     │                  │  BLOCKED     │ → Telegram + Linear
        │     │                  └──────────────┘
        │     │
        ┌─────▼────────────┐
        │  SECURITY_REVIEW  │ ← Opus 4.6 reviewing
        └─────┬────────────┘
              │
        ┌─────┼──────────┐
        │     │           │
    (approved) │    (issues)
        │     │           │
        │     │     ┌─────▼────────┐
        │     │     │  SEC_FIXING  │ ← fixing security issues
        │     │     └─────┬────────┘
        │     │           │
        │     │     ┌─────▼────────┐
        │     │     │  CI_RUNNING   │ ← CI must re-run after fix!
        │     │     └─────┬────────┘
        │     │           │
        │     │     (pass)│→ SECURITY_REVIEW (re-review)
        │     │           │
        │     │     (CRITICAL finding at any point)
        │     │           │
        │     │     ┌─────▼────────┐
        │     │     │  BLOCKED     │ ← CRITICAL + security:blocked label
        │     │     └──────────────┘
        │     │
        ┌─────▼────────┐
        │  MERGING      │ ← auto-merge gate
        └─────┬────────┘
              │
        ┌─────▼────────┐
        │    DONE       │ ← merged to main
        └──────────────┘

BLOCKED state exits (human-driven):
  BLOCKED → QUEUED     (human re-opens, PO re-decomposes)
  BLOCKED → CANCELLED  (human decides task is not needed)

SPLIT parent aggregation:
  All sub-tasks DONE       → parent DONE
  Any sub-task BLOCKED     → parent BLOCKED (escalation)
  Any sub-task CANCELLED   → PO reassesses remaining sub-tasks
```

---

## 20. Sequence Diagrams

### Happy Path Sequence

```
Linear    Orchestrator    PO       Dev      GitHub    CI      QA    Security   Marian
  │            │           │        │         │        │       │        │        │
  │──webhook──▶│           │        │         │        │       │        │        │
  │            │──dispatch─▶│       │         │        │       │        │        │
  │            │           │──analyze──┐      │        │       │        │        │
  │            │           │◀──result──┘      │        │       │        │        │
  │◀─status────│◀──assign──│        │         │        │       │        │        │
  │            │──────────dispatch──▶│        │        │       │        │        │
  │            │           │        │──code───▶│       │        │       │        │
  │            │           │        │         │──push──▶│      │        │        │
  │            │           │        │──PR────▶│        │       │        │        │
  │            │           │        │         │        │──run──▶│       │        │
  │            │           │        │         │        │       │──monitor│       │
  │            │           │        │         │        │◀──pass─│       │        │
  │            │           │        │         │        │       │──trigger▶       │
  │            │           │        │         │        │       │        │──review│
  │            │           │        │         │        │       │◀─approved       │
  │            │           │        │         │◀─merge─│       │        │        │
  │◀─done──────│           │        │         │        │       │        │        │
  │            │───────────────────────────────────────────────────notify▶       │
  │            │           │        │         │        │       │        │    ──▶Telegram
```

### CI Retry Sequence

```
Dev      GitHub    CI      QA       Dev(fix)   CI(retry)
 │         │       │        │          │          │
 │──push──▶│       │        │          │          │
 │         │──run──▶│       │          │          │
 │         │       │──FAIL──▶│         │          │
 │         │       │        │──read log│          │
 │         │       │        │──classify│          │
 │         │       │        │──dispatch▶│         │
 │         │       │        │          │──fix     │
 │         │       │        │          │──push───▶│
 │         │       │        │          │          │──run
 │         │       │        │◀─────────│──────────│──PASS
 │         │       │        │──proceed to security review
```

---

## 21. Configuration Reference

### Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `LINEAR_API_KEY` | Yes | Linear API key for OPE team | `lin_api_xxx` |
| `LINEAR_WEBHOOK_SECRET` | Yes | Webhook signature verification | `whsec_xxx` |
| `LINEAR_TEAM_ID` | Yes | Linear team identifier | `OPE` |
| `OPENROUTER_API_KEY` | Yes | OpenRouter API key | `sk-or-xxx` |
| `GITHUB_TOKEN` | Yes | GitHub token (repo + PR scope) | `ghp_xxx` |
| `GITHUB_REPO` | Yes | Target repository | `openoms-org/openoms` |
| `REDIS_URL` | Yes | Redis connection string | `redis://redis:6379` |
| `TELEGRAM_BOT_TOKEN` | Yes | Telegram bot token | `123456:ABC-xxx` |
| `TELEGRAM_CHAT_ID` | Yes | Telegram chat for notifications | `-100xxx` |
| `MARIAN_URL` | No | Marian service URL | `http://marian:3000` |
| `LOG_LEVEL` | No | Log level (debug/info/warn/error) | `info` |
| `MAX_CI_RETRIES` | No | Max CI retry attempts per PR | `5` |
| `BUDGET_MONTHLY_LIMIT` | No | Monthly budget cap in USD | `280` |
| `WEBHOOK_PORT` | No | Port for webhook receiver | `8090` |
| `POLL_INTERVAL_SECONDS` | No | Linear polling interval | `300` |

### Linear Workflow States

| State | When | Set By |
|-------|------|--------|
| Backlog | Ticket created, not prioritized | Human |
| Todo | Prioritized, ready for agent | Human |
| In Progress | PO assigned, agent working | PO Agent |
| In Review | PR created, CI running | Dev Agent |
| Done | PR merged | Orchestrator |
| Blocked | Agent stuck, needs human | QA / PO Agent |
| Cancelled | Superseded or no longer needed | Human / PO |

### Linear Labels

| Label | Purpose |
|-------|---------|
| `bot:backend-dev` | Assigned to backend dev agent |
| `bot:frontend-dev` | Assigned to frontend dev agent |
| `bot:integration-dev` | Assigned to integration dev agent |
| `bot:devops` | Assigned to devops agent |
| `manual` | Skip — Rafal handles manually |
| `migration:destructive` | Has destructive DB changes — human approval required |
| `risk:security` | Touches auth/security code |
| `risk:billing` | Touches billing code |
| `complexity:simple` | 1-2 files |
| `complexity:medium` | 3-5 files |
| `complexity:complex` | 6+ files (decomposed) |

---

## 22. Open Questions & Risks

### Open Questions

1. **NemoClaw beta stability:** NemoClaw is Alpha (announced GTC 2026-03-06). Fallback plan defined (plain Docker + OpenClaw), but when exactly do we cut over? After 2 weeks of testing? After first production incident?

2. **OpenRouter Nemotron availability:** Nemotron 120B on OpenRouter — is it reliably available 24/7? Fallback chain defined (120B → 70B → Nano 30B → Sonnet), but should we pre-provision a self-hosted instance as insurance?

3. **GitHub App vs PAT:** Should we use a GitHub App (higher rate limits, fine-grained permissions, auto-rotating tokens) instead of a PAT? Spec currently assumes PAT. GitHub App is more secure but requires more setup. **Recommendation: GitHub App** — the security-bot account for PR reviews (C-4 fix) works better as a GitHub App.

4. **Kilo Code coexistence:** When Rafal works manually, he must add `manual` label in Linear. What if he forgets? Should the orchestrator detect human branches (non-bot/ prefix) touching same files and auto-pause?

5. **PO bottleneck throughput:** PO processes tasks sequentially (1 at a time). At 30s-2min per decomposition, burst of 10 tickets = 5-20 min queue time. Is batch-processing PO calls (multiple tickets in one LLM call) safe? Or should QA bypass PO for simple fix-cycle re-dispatches?

6. **Code quality reviewer gap:** Spec has Security Reviewer but no general code quality reviewer. CodeRabbit partially fills this role. Is CodeRabbit sufficient, or do we need a dedicated quality reviewer agent (using existing `reviewer.md` definition)?

### Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| NemoClaw instability | Medium | High | Fallback: run agents as plain Docker containers with OpenClaw |
| Agents introduce bugs | High | Medium | CI (11 jobs) + Security Reviewer (PR review) + CodeRabbit |
| Token costs exceed budget | Medium | Low | Hard budget cap + alerts + model fallback chain |
| OpenRouter downtime | Low | High | Fallback model chain + task queuing |
| Agent creates security vulnerability | Medium | Critical | Mandatory Opus 4.6 security review + NEVER rules + sandbox |
| Merge conflict storms | Low | Medium | PO file-overlap prevention + serialized coordination files |
| Context window overflow | Medium | Medium | Proactive compaction at 70% + token budget per agent |
| Infinite agent loops | Medium | Low | Loop detection heuristics + max iteration limits |
| Redis data loss | Low | High | AOF persistence + daily RDB backups |

---

## Appendix A: API Integration Details

### Linear GraphQL API

**Endpoint:** `https://api.linear.app/graphql`

**Create sub-issues** (requires special header):
```graphql
# Header: GraphQL-Features: sub_issues
mutation addSubIssue {
  addSubIssue(input: { issueId: "PARENT_ID", subIssueId: "CHILD_ID" }) {
    issue { title }
    subIssue { title }
  }
}
```

**Query workflow states for OPE team:**
```graphql
query {
  workflowStates(filter: { team: { id: { eq: "OPE_TEAM_ID" } } }) {
    nodes { id name type }
  }
}
```

**Webhook payload includes previous values** — enables tracking state transitions:
```json
{
  "action": "update",
  "type": "Issue",
  "data": { "stateId": "new_state", "title": "..." },
  "updatedFrom": { "stateId": "old_state" }
}
```

**Webhook security:** HMAC-SHA256 signature verification with LINEAR_WEBHOOK_SECRET.

### GitHub GraphQL — Unresolved Review Threads

The auto-merge gate needs to check CodeRabbit threads. REST API lacks resolved/unresolved state — must use GraphQL:

```graphql
query {
  repository(owner: "openoms-org", name: "openoms") {
    pullRequest(number: $prNumber) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 1) {
            nodes { author { login } body }
          }
        }
      }
    }
  }
}
```

Filter for `author.login == "coderabbitai"` and `isResolved == false`.

### OpenRouter Model Fallback Configuration

```json
{
  "model": "nvidia/nemotron-3-super-120b-a12b",
  "models": [
    "nvidia/nemotron-3-super-120b-a12b",
    "nvidia/nemotron-3-super-70b",
    "nvidia/nemotron-3-nano-30b-a3b",
    "anthropic/claude-sonnet-4-6"
  ],
  "provider": {
    "order": ["NVIDIA", "Together"],
    "allow_fallbacks": true
  }
}
```

OpenRouter's built-in fallback: if first model in `models` array returns error, tries the next. Billing is for the model that ultimately succeeds.

### NemoClaw CLI Reference

```bash
nemoclaw setup    # Full setup: gateway + providers + sandbox
nemoclaw connect  # Interactive shell inside sandbox
nemoclaw status   # Health, blueprint state, inference config
nemoclaw logs     # Stream blueprint + sandbox logs
```

Install: `curl -fsSL https://docs.nvidia.com/nemoclaw/install.sh | bash`
Architecture: TypeScript plugin (OpenClaw CLI) + Python blueprint (OpenShell orchestration)
Sandbox: Landlock + seccomp + network namespaces, `/sandbox` and `/tmp` only

---

## Appendix B: Glossary

| Term | Definition |
|------|-----------|
| NemoClaw | NVIDIA OpenClaw plugin for sandboxed agent orchestration (NOT a fork — a security layer wrapping OpenClaw) |
| OpenShell | NemoClaw's sandbox runtime |
| OpenClaw | Open-source agent framework (Marian runs on this) |
| Marian | OpenClaw agent that relays status to Telegram/WhatsApp |
| PO | Product Owner agent — decomposes and assigns tickets |
| Auto-merge gate | CI job that checks all required checks + CodeRabbit before merging bot/ PRs |
| CodeRabbit | AI code review tool (free for public repos) |
| Linear | Task management SaaS (team key: OPE) |
| OpenRouter | API gateway for multiple LLM providers |
| Nemotron 3 Super 120B | NVIDIA MoE model, 120B params, 12B active, 1M context |
| Repo map | tree-sitter generated index of all files/symbols in the repo |
| Work package | Structured JSON message from PO to Dev agent with full task context |

---

## Appendix C: Research Sources

- [NVIDIA NemoClaw GitHub](https://github.com/NVIDIA/NemoClaw) + [Docs](https://docs.nvidia.com/nemoclaw/latest/)
- [OpenClaw Official](https://openclaw.ai/) + [ACP Docs](https://docs.openclaw.ai/tools/acp-agents)
- [OpenRouter Rate Limits](https://openrouter.ai/docs/api/reference/limits) + [Model Fallbacks](https://openrouter.ai/docs/guides/routing/model-fallbacks)
- [Linear Webhooks](https://linear.app/developers/webhooks) + [GraphQL API](https://linear.app/developers/graphql)
- [GitHub Auto-merge](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request)
- [Cognition: Don't Build Multi-Agents](https://cognition.ai/blog/dont-build-multi-agents)
- [Microsoft Taxonomy of Agent Failure Modes](https://www.microsoft.com/en-us/security/blog/2025/04/24/new-whitepaper-outlines-the-taxonomy-of-failure-modes-in-ai-agents/)
- [Anthropic: Effective Context Engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents)
- [Aider: Repository Maps](https://aider.chat/docs/repomap.html)
- [IBM: STRATUS Undo-and-Retry](https://research.ibm.com/blog/undo-agent-for-cloud)
- [2026 Agentic Coding Trends (Anthropic)](https://resources.anthropic.com/2026-agentic-coding-trends-report)

---

## Appendix D: Decision Log

| # | Decision | Rationale | Date |
|---|----------|-----------|------|
| 1 | Use NemoClaw over plain OpenClaw | Sandboxing, RBAC, audit logs — critical for autonomous agents | 2026-03-17 |
| 2 | Nemotron 120B for dev agents | 10x cheaper than Opus, 1M context, sufficient for code generation | 2026-03-17 |
| 3 | Opus 4.6 for security + CEO only | Higher reasoning capability needed for security analysis | 2026-03-17 |
| 4 | Linear for task management | Already in use, proper API, not GitHub Issues | 2026-03-17 |
| 5 | Max 1 agent per role concurrently | Prevents file conflicts, simplifies state management | 2026-03-17 |
| 6 | 5 CI retries max | Balances persistence vs token waste | 2026-03-17 |
| 7 | Redis for state | Already deployed, fast, AOF for persistence | 2026-03-17 |
| 8 | Repo map via tree-sitter | Proven approach (aider), massive token savings | 2026-03-17 |
| 9 | bot/ branch prefix | Identifies agent PRs, triggers auto-merge gate | 2026-03-17 |
| 10 | Squash merge only for agents | Clean history, single commit per ticket | 2026-03-17 |
