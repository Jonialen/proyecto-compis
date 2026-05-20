# Skill Registry — proyecto-compis

Generated: 2026-05-10
Project: proyecto-compis
Stack: Go 1.26 (module: genanalex)

---

## Active Skills

### Go & Testing

| Skill | Trigger | Source |
|-------|---------|--------|
| `go-testing` | When writing Go tests, using teatest, or adding test coverage | user-level |
| `work-unit-commits` | When implementing a change, preparing commits, splitting PRs, or planning chained/stacked PRs | user-level |

### Collaboration & Review

| Skill | Trigger | Source |
|-------|---------|--------|
| `comment-writer` | When drafting or posting feedback, review comments, maintainer replies, Slack messages, or GitHub comments | user-level |
| `branch-pr` | When creating a pull request, opening a PR, or preparing changes for review | user-level |
| `issue-creation` | When creating a GitHub issue, reporting a bug, or requesting a feature | user-level |
| `judgment-day` | When user says "judgment day", "review adversarial", "dual review", "juzgar", "que lo juzguen" | user-level |
| `chained-pr` | When a PR would exceed 400 changed lines, when planning chained/stacked PRs | user-level |
| `cognitive-doc-design` | When writing guides, READMEs, RFCs, onboarding docs, or architecture docs | user-level |

### SDD Workflow

| Skill | Trigger | Source |
|-------|---------|--------|
| `sdd-init` | When user wants to initialize SDD in a project | user-level |
| `sdd-explore` | When the orchestrator launches to think through a feature or investigate the codebase | user-level |
| `sdd-propose` | When the orchestrator launches to create or update a proposal | user-level |
| `sdd-spec` | When the orchestrator launches to write or update specs | user-level |
| `sdd-design` | When the orchestrator launches to write or update technical design | user-level |
| `sdd-tasks` | When the orchestrator launches to create or update task breakdown | user-level |
| `sdd-apply` | When the orchestrator launches to implement tasks | user-level |
| `sdd-verify` | When the orchestrator launches to verify a completed change | user-level |
| `sdd-archive` | When the orchestrator launches to archive a completed change | user-level |
| `sdd-onboard` | When the orchestrator launches to onboard a user through the full SDD cycle | user-level |

### Meta

| Skill | Trigger | Source |
|-------|---------|--------|
| `skill-creator` | When user asks to create a new skill or add agent instructions | user-level |
| `skill-registry` | When user says "update skills", "skill registry", "actualizar skills", or "update registry" | user-level |

---

## Project Conventions

- No project-level AGENTS.md / CLAUDE.md / .cursorrules detected
- No CI/CD workflows detected (.github/ absent)
- No linter config detected (golangci-lint not configured)
- Testing: `go test ./...` — native Go test runner
- Coverage: `go test -cover ./...`
- Format: `gofmt`
- Vet: `go vet ./...`
