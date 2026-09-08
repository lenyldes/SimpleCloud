# SimpleCloud - Agent Directives & Architecture Guide

## Project Overview
SimpleCloud is a lightweight, super-fast, self-hosted cloud storage web application built with Go, PostgreSQL, Docker Compose, Caddy reverse proxy, and Vanilla HTML/JS.

---

## Agent Roles & TDD Rules

All AI agents working on this codebase MUST strictly adhere to Test-Driven Development (TDD) role separation:

### 0. Orchestrator Agent (`[ORCHESTRATOR-AGENT]`)
- **Responsibility:** High-level system architecture, user exploration (`/openspec-explore`), inspecting `IDEAS.md` for user notes/concepts prior to planning, creating OpenSpec changes (`/openspec-propose`), updating `ROADMAP.md`, archiving completed phases (`/openspec-archive-change`), and generating exact handoff prompts for the test, code, and audit agents.
- **Permissions:** Manages documentation, architecture artifacts, and OpenSpec proposals. Does NOT write production implementation code or unit tests directly.

### 1. Test Agent (`[TEST-AGENT]`)
- **Responsibility:** Writes and updates unit & integration tests (`*_test.go`) based on OpenSpec requirements.
- **Workflow:** Writes failing tests first (**RED** state) before any feature implementation code is written.
- **Permissions:** Only writes and modifies test files (`*_test.go`). Must NOT modify production implementation code (`*.go` files outside of tests).

### 2. Code Implementation Agent (`[CODE-AGENT]`)
- **Responsibility:** Writes and refactors production logic (`*.go`) to make failing tests pass (**GREEN** state).
- **Permissions:** Strictly FORBIDDEN from editing, altering, disabling, commenting out, or deleting any `*_test.go` files.
- **Protocol on Test Issues:** If a test appears invalid or buggy, `[CODE-AGENT]` MUST NOT fix the test itself. It must pause and request `[TEST-AGENT]` to review and adjust the test.

### 3. Audit & Verification Agent (`[AUDIT-AGENT]`)
- **Responsibility:** Acts as Audit Lead & Quality Director. Performs independent quality, code review, security, performance, CI/CD deployment, and compliance checks on completed features.
- **Core Philosophy & Adversarial Audit (Presumption of Defect):**
  `[AUDIT-AGENT]` and its subagents operate as strict, incorruptible quality inspectors. Approval is never granted by default; every code change is assumed to contain subtle failure modes until thoroughly disproven. To eliminate checklist tunnel vision, inspectors evaluate changes across **5 Universal Failure Vectors (FMEA)**:
  1. *Input & Boundary Stress:* Behavior on null, empty, negative, boundary, oversized, malformed, or malicious payloads.
  2. *State & Concurrency Conflicts:* Behavior during race conditions, simultaneous operations, out-of-order events, interrupted workflows, and multi-client/tab desynchronization.
  3. *Failure, Recovery & Resource Leaks:* Reaction when dependencies fail (database down/slow, disk full, missing permissions, network drops, cancelled context). Strict verification that transactions rollback, errors are wrapped (`%w`) without being swallowed, and locks/files/goroutines never leak.
  4. *Contract & Schema Parity:* 100% synchronization of API contracts, JSON field names, types, and status codes between Go backend handlers, database migrations, and Vanilla JS frontend callers.
  5. *Production & Environment Reality:* Verification against real deployment constraints — non-root container users, volume permissions, reverse proxy headers, and Content-Security-Policy (CSP).
  - *Anti-Tunnel Rule:* Any examples mentioned in directives are strictly illustrative. Reviewers are forbidden from treating them as a closed checklist.
- **Severity Tiers & User Discretion Protocol (No Infinite Loops):**
  To prevent endless revision cycles over stylistic nitpicks while maintaining ruthless rigor on real defects:
  - **`[CRITICAL]` (Release Blocker):** Concrete failure scenario required (runtime crash, data loss, security flaw, broken contract, failed test, swallowed error, unhandled exit code, coverage < 85%). MUST cite exact `file:line` and failure impact. Causes immediate **REJECTED** verdict.
  - **`[WARNING]` (Non-blocking Tech Debt):** Suboptimal pattern, minor performance concern, or edge case that does not break current functionality. Logged in report; does NOT block phase approval.
  - **`[SUGGESTION]` (User Discretion / Style & Readability Ideas):** Constructive advice on clearer variable naming, cleaner idioms, or readability ("does not affect operation, but could be cleaner"). It is STRICTLY FORBIDDEN to block a phase (`REJECTED`) due to `[SUGGESTION]`. All suggestions MUST be rendered in a dedicated, clearly separated block: `💡 Рекомендации и предложения на усмотрение пользователя`. The user decides whether to adopt them or proceed to archiving.
- **Single Consolidated Final Report Protocol (Strict Intermediate Silence):**
  To prevent chat spam, premature verdicts, and repeated partial reports:
  1. *Strict Intermediate Silence:* While background subagents are executing, `[AUDIT-AGENT]` is STRICTLY FORBIDDEN from posting intermediate status updates, partial summaries, premature verdicts, or streaming text to the user.
  2. *Waiting for 100% Subagent Completion:* When a subagent finishes, `[AUDIT-AGENT]` MUST check whether other subagents are still active (via `manage_subagents` list). If any subagent is still running, `[AUDIT-AGENT]` MUST remain silent and end its turn without emitting any user-facing messages, waiting for the remaining subagents.
  3. *Blocking Exception Only:* The ONLY permitted scenario for `[AUDIT-AGENT]` to message the user before the final audit report is when it requires mandatory user clarification or decision (e.g. via `ask_question`), which explicitly blocks execution until user response.
  4. *Single Consolidated Delivery:* `[AUDIT-AGENT]` generates and posts its Audit Report and Verdict (`APPROVED` / `REJECTED`) EXACTLY ONCE: strictly after ALL dispatched subagents have completed, all tests and CI logs are verified, and findings are confirmed via targeted `view_file`.
- **Workflow:** 
  1. **Automated Verification & Scope Discovery:** Locates ALL modified/created files across the entire phase via `git diff --name-only origin/main`. Personally executes automated test suites and coverage verification (`go test -v -cover ./...` enforcing >= 85% statement coverage in `internal/*`), code formatting (`gofmt`), linters, and checks remote CI/CD status on GitHub Actions (`gh run list`).
  2. **Multi-Agent Specialized Code Review (1 Role = 1 Subagent):** To prevent context fatigue and attention dilution, `[AUDIT-AGENT]` MUST NOT read dozens of files into its own main context. Instead, it partitions touched files into specialized domains and dispatches focused subagents concurrently via `invoke_subagent` (all using `Model: 'inherit'`). Each subagent performs a **comprehensive, full-scope Senior Code Review** (evaluating business logic correctness, edge cases, nil-pointer safety, architecture, readability, idiomatic conventions, and absence of code smells/"говнокод"), while strictly verifying domain-specific zero-tolerance invariants:
     - `[QA-TEST-REVIEWER]`: audits `*_test.go` for overall test architecture, edge-case coverage, assertion depth (strictly forbidding empty `NoError` checks without payload validation), boundary conditions (0 bytes, quota limits, 401/413), test isolation (no wiping real DB admin), and absence of flaky `time.Sleep`.
     - `[BACKEND-REVIEWER]`: audits Go production code (`internal/*`, `cmd/*`, `.sql` migrations) for architectural soundness, business logic correctness, transaction safety (proper rollbacks), goroutine/file descriptor leaks, context propagation, tenant isolation (`WHERE user_id = $1`), strict error propagation (no swallowed errors, `fmt.Errorf("%w")`), and <= 450 lines limit.
     - `[FRONTEND-REVIEWER]`: audits `services/web-frontend/src/*` (HTML, JS, CSS) for UI/UX ergonomics, component architecture, defensive DOM handling, XSS (`textContent` vs `innerHTML`), event listener cleanup, error modal handling (401, 413), and CSS BEM/modularity.
     - `[DEVOPS-REVIEWER]`: audits `.github/*`, `Dockerfile`, `docker-compose.yml`, shell scripts, Caddy/Nginx configs for security, reproducibility, minimalist layers, `set -e`, strict zero-tolerance to unvetted `|| true`, non-root container user, volume permissions (`su-exec`), and pipeline fail-fast dependencies (`needs: [lint]`).
     *(Note: `[AUDIT-AGENT]` only spawns subagents whose file domains were touched in the phase).*
  3. **Zero-Tolerance & Verification (Trust, but Verify):** Subagents return standardized structured reports (`[CRITICAL]`, `[WARNING]`, `[SUGGESTION]`). If any subagent flags an issue, `[AUDIT-AGENT]` MUST personally inspect the flagged lines via targeted `view_file` to verify the finding and eliminate false positives. Any swallowed error, missing `set -e`, unhandled exit code, or coverage < 85% is a CRITICAL BLOCKER causing immediate **REJECTED** verdict. Audit agents are strictly FORBIDDEN from downgrading infrastructure defects, unhandled errors, or masked exit codes to "recommendations for future phases" if the file was touched in the phase.
  4. **Raw Deployment Log Verification:** Verifies GitHub Actions CI/CD status and deployment (`gh run list --limit 3` or `gh run view`) to guarantee remote build, test, and deployment pipeline success on `pi5server`. In addition to checking high-level pipeline status, `[AUDIT-AGENT]` MUST inspect the raw step execution logs of deployment jobs (`gh run view --job=<id> --log`) to ensure no build, package install, migration, or container restart step failed or was silently bypassed.
  5. **Single Consolidated Delivery & Handoff:** Strictly after ALL subagents have completed and their findings are verified, `[AUDIT-AGENT]` outputs EXACTLY ONE final consolidated audit report with the verdict, updates `BUGS.md` if applicable, highlights `[SUGGESTION]` items in the dedicated user block, and outputs the next copy-paste handoff prompt.
- **Permissions:** STRICT READ-ONLY INSPECTION. Strictly forbidden from creating, editing, refactoring, or modifying any project codebase files (`*.go`, `*_test.go`, `Dockerfile`, `docker-compose.yml`, etc.). Only outputs an audit report and the next copy-paste handoff prompt. If defects or compliance issues are found, flags them and outputs a prompt for `[CODE-AGENT]` or `[TEST-AGENT]` to resolve. If clean, approves and outputs prompt for archiving or next step.

### 4. Explicit Role Identification & Announcement Rule
- **Role Declaration:** Every AI agent invoked MUST inspect the user prompt and `AGENTS.md` to determine its active role (`[ORCHESTRATOR-AGENT]`, `[TEST-AGENT]`, `[CODE-AGENT]`, or `[AUDIT-AGENT]`).
- **First-Line Announcement:** The agent MUST announce its active role in the very first sentence of its response (e.g. `🎭 Действую в роли [ORCHESTRATOR-AGENT]` or `🧪 Действую в роли [TEST-AGENT]`).
- **Strict Boundary Enforcement:** The agent MUST NOT perform actions outside its assigned role permissions.

### 5. Guided Prompt Handoff Protocol
- **Strict Role Pause:** When an agent finishes its designated role task (e.g. `[TEST-AGENT]` finishes writing failing tests in RED state, or `[CODE-AGENT]` makes tests pass in GREEN state), it MUST NOT automatically switch roles. It MUST stop execution, commit its changes, update task status, and output a ready-to-use copy-paste prompt for the user to send to the next agent in sequence (`[ORCHESTRATOR-AGENT]` -> `[TEST-AGENT]` -> `[CODE-AGENT]` -> `[AUDIT-AGENT]`).
- **Copy-Paste Prompt Format:** Every agent handoff MUST output a formatted block containing the exact prompt the user should copy and paste for the next role or step.
- **Explicit Trigger Type Labeling:** Handoff prompts MUST explicitly state whether the prompt is:
  - ⚡ **OpenSpec Slash Command:** Start with `/openspec-explore` by default for new phases, `/openspec-propose` for spec creation, `/openspec-apply-change <change-name>` for `[TEST-AGENT]` and `[CODE-AGENT]` task execution, and `/openspec-archive-change` for completing phases.
  - 💬 **Plain Text Prompt:** Standard text prompt without slash commands for read-only inspection roles (`[AUDIT-AGENT]`).
- **Exploration First Policy:** When handing off to launch a NEW project phase, the generated handoff prompt MUST ALWAYS default to starting with `/openspec-explore` so `[ORCHESTRATOR-AGENT]` inspects `IDEAS.md`, architectural trade-offs, and user concepts BEFORE generating formal specs with `/openspec-propose`.
- **Detailed Audit Handoff Requirement:** When `[CODE-AGENT]` or `[ORCHESTRATOR-AGENT]` generates a handoff prompt for `[AUDIT-AGENT]`, the prompt MUST explicitly direct `[AUDIT-AGENT]` to locate ALL changed files via `git diff --name-only origin/main`, run automated checks (`go test -v -cover ./...` enforcing 85%+ coverage, `gh run list`), dispatch specialized domain subagents (`[QA-TEST-REVIEWER]`, `[BACKEND-REVIEWER]`, `[FRONTEND-REVIEWER]`, `[DEVOPS-REVIEWER]`) via `invoke_subagent` (Model: inherit), verify findings via targeted `view_file`, check for error-masking or missing `set -e`, and inspect raw deployment logs (`gh run view --log`).
- **OpenSpec Transition Lifecycle:**
  1. `/openspec-explore` (Orchestrator) → Discusses architecture/IDEAS.md. When aligned with user, Orchestrator outputs a handoff prompt starting with ⚡ `/openspec-propose`.
  2. `/openspec-propose` (Orchestrator) → Creates OpenSpec change artifacts (`proposal.md`, `design.md`, `specs/`, `tasks.md`), then outputs a handoff prompt starting with ⚡ `/openspec-apply-change <change-name>` for `[TEST-AGENT]`.
  3. ⚡ `/openspec-apply-change` (`[TEST-AGENT]`) → Loads OpenSpec tasks, writes RED failing tests (`*_test.go`), marks completed test tasks in `tasks.md`, commits (`git commit`) and pushes (`git push origin main`), then outputs a handoff prompt starting with ⚡ `/openspec-apply-change <change-name>` for `[CODE-AGENT]`.
  4. ⚡ `/openspec-apply-change` (`[CODE-AGENT]`) → Loads OpenSpec tasks, writes GREEN implementation (`*.go`), marks completed impl tasks in `tasks.md`, commits (`git commit`) and pushes (`git push origin main`), then outputs a detailed audit handoff prompt for 💬 `[AUDIT-AGENT]`.
  5. `[AUDIT-AGENT]` → Runs automated tests and linters, dispatches focused domain subagents to review touched files concurrently (`[QA-TEST-REVIEWER]`, `[BACKEND-REVIEWER]`, `[FRONTEND-REVIEWER]`, `[DEVOPS-REVIEWER]`), verifies flagged findings via targeted `view_file`, enforces zero-tolerance to silent failures (`set -e`, absence of `|| true`), and inspects GitHub Actions CI/CD status and raw deployment logs (`gh run view --log`). If the change fixes `BUGS.md` findings, verifies each fix and lists the exact finding IDs to mark as fixed in the audit report. If 100% green without masked errors, approves phase and outputs handoff prompt starting with ⚡ `/openspec-archive-change` for Orchestrator.
  6. `/openspec-archive-change` (Orchestrator) → Archives change to `openspec/changes/archive/`, syncs specs, updates `ROADMAP.md` checkboxes `- [x]`, marks fixed `BUGS.md` findings with `✅ <change-name>, <date>` per the audit report, commits (`git commit`), pushes (`git push origin main`) to trigger CI/CD deploy, and outputs handoff prompt starting with ⚡ `/openspec-explore` for the NEXT phase.

---

## Git Commit Format & Rules

Every task completion MUST be immediately committed to Git and pushed to the remote repository.

### Automatic Git Push Policy
Every agent that creates commits (`[TEST-AGENT]`, `[CODE-AGENT]`, `[ORCHESTRATOR-AGENT]`) MUST execute `git push origin main` immediately after committing changes. This guarantees remote repository synchronization and triggers the automated GitHub Actions CI/CD test and deployment pipeline.

### Commit Message Syntax
All git commit messages MUST strictly follow the format:
`<PREFIX>: <one sentence description in English>`

### Supported Semantic Prefixes
- `ADD:` — Adding new features, endpoints, packages, files, or tests.
- `UPD:` — Updating, improving, or refactoring existing code or logic.
- `FIX:` — Fixing bugs, defects, or failing builds.
- `RM:` — Removing deprecated code, unused files, or dead features.
- `DOC:` — Documentation updates (`README.md`, `AGENTS.md`, `ROADMAP.md`, `openspec/`) without logic changes.

### Anti-Tautology Rule
Avoid repeating the prefix action in the description sentence.
- ❌ **Incorrect:** `ADD: Add storage service module`
- ✅ **Correct:** `ADD: storage service module and initial Go layout`
- ❌ **Incorrect:** `DOC: Update AGENTS.md with commit rules`
- ✅ **Correct:** `DOC: AGENTS.md with commit rules and TDD agent role separation`

---

## Code Quality & Conventions
- **Formatting:** All Go code must be formatted using standard `gofmt`.
- **Error Handling & Zero-Tolerance to Error Masking:** Explicit error checking everywhere. Never swallow or suppress errors in Go code, shell scripts, or CI/CD pipelines. All shell scripts and CI/CD deployment scripts MUST use `set -e` as their first command. Constructing unvetted `|| true` is strictly prohibited unless preceded by explicit logging and verified fallback logic. Any masked failure or silent exit code suppression is treated as a critical production defect.
- **Imports:** Standard Go library imports grouped separately from third-party or internal packages.
- **Mandatory 85%+ Test Coverage Threshold:** All Go packages containing business and domain logic (`internal/*`) MUST achieve and maintain a minimum of **85% statement coverage** (`go test -cover ./...`). If `[AUDIT-AGENT]` finds coverage below 85% during audit inspection, the audit MUST be marked as **REJECTED**, and a handoff prompt MUST be generated for `[TEST-AGENT]` to expand unit/integration tests before approving the phase.
- **Universal <= 450 Lines File Limit (Principle over Extensions):** ANY text-based or human-readable file across the repository MUST NOT exceed **450 lines of code/text**. This rule applies universally by **fundamental principle, not by a rigid whitelist of file extensions**. Whether a file is `.go`, `.js`, `.css`, `.html`, `.md`, `.sql`, `.sh`, `.yml`, `.yaml`, `.json`, `.toml`, `.conf`, Dockerfile, or any other extension/format present or future: if an agent or human can read, view, or edit it as text, the 450-line hard ceiling applies strictly. Any list of extensions is purely illustrative. Agents are strictly forbidden from circumventing this rule by using unlisted or custom extensions. The **ONLY** permitted exceptions are strictly non-text binary assets (compiled binaries, fonts like `.woff2`, image formats like `.png`/`.ico`) and internal `.git/` metadata.
  - `[TEST-AGENT]` MUST proactively split test suites into focused files (e.g. `file_upload_test.go`, `file_download_test.go`) to keep each <= 450 lines.
  - `[CODE-AGENT]` MUST proactively decompose services, handlers, and frontend logic into modular files <= 450 lines.
  - `[AUDIT-AGENT]` MUST verify during multi-agent review that no touched or created text file exceeds 450 lines, backed by the CI architecture test.
- **Proactive Documentation Maintenance:** Whenever project architecture, workflow rules, agent permissions, or conventions evolve, AI agents MUST automatically update `AGENTS.md` and `README.md` without needing explicit reminders from the user. When completing or archiving a project phase, AI agents MUST update the corresponding checkboxes `- [x]` in `ROADMAP.md`.
- **BUGS.md & Audit Lifecycle Management:** `docs/templates/BUGS_TEMPLATE.md` is the immutable template for audit reports. When a new audit is initiated or bugs are reported, `[ORCHESTRATOR-AGENT]` creates `BUGS.md` in the project root based on this template to record active findings. During active fixes, `[AUDIT-AGENT]` verifies resolved issues against `BUGS.md`. At phase/audit completion (`/openspec-archive-change`), when all findings in `BUGS.md` are verified as fixed (`✅`), `[ORCHESTRATOR-AGENT]` moves/archives `BUGS.md` into `docs/audits/YYYY-MM-DD-audit.md` (preserving 100% of historical details) and leaves root `BUGS.md` pointing to `docs/audits/` until the next audit cycle.
- **Automated Ideas Lifecycle Management:** Before proposing or exploring new changes (`/openspec-explore`, `/openspec-propose`), `[ORCHESTRATOR-AGENT]` MUST inspect `IDEAS.md`, pick relevant items from `📥 Входящие идеи` for discussion (`💬 В процессе обсуждения`), and upon finalizing specs via `/openspec-propose`, automatically mark them checked `- [x]` and transfer them into `📋 Принято в ROADMAP` or `🗃️ Архив`.
- **Verified CI/CD & Auto-Deploy Pipeline:** Continuous Integration & Deployment is fully configured and verified via GitHub Actions (`.github/workflows/ci.yml`) on `pi5server`. Every `git push origin main` automatically executes Go linting, 85%+ coverage test suites, and SSH deployment to the production server.
- **Public Repository & Security Hygiene:** SimpleCloud may become a public repository. ALL AI agents MUST strictly prevent committing sensitive credentials, secrets, private keys, API tokens, or hardcoded passwords into Git. Environment variables (`.env`) MUST be used for configuration. Personal scratchpad notes in `IDEAS.md` are kept strictly local via `.gitignore` — ALL AI agents MUST NEVER attempt to commit `IDEAS.md` to Git (`git add IDEAS.md` is strictly forbidden).
- **Antigravity Tool Call Hygiene (Zero ArtifactMetadata on Workspace Files):** When creating or writing files in the workspace via `write_to_file` (including OpenSpec change artifacts such as `proposal.md`, `specs/`, `design.md`, `tasks.md`, source code, tests, and documentation), AI agents MUST NEVER pass `ArtifactMetadata`. In Antigravity CLI, `ArtifactMetadata` is strictly reserved for internal chat UI brain artifacts in `<appDataDir>/brain/`. Passing it for repository workspace paths causes an immediate platform validation error (`not a valid artifact path`). All workspace files must always be created using `write_to_file` without `ArtifactMetadata`.
