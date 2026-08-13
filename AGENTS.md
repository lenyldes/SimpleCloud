# SimpleCloud - Agent Directives & Architecture Guide

## Project Overview
SimpleCloud is a lightweight, super-fast, self-hosted cloud storage web application built with Go, PostgreSQL, Docker Compose, Caddy reverse proxy, and Vanilla HTML/JS.

---

## Agent Roles & TDD Rules

All AI agents working on this codebase MUST strictly adhere to Test-Driven Development (TDD) role separation:

### 0. Orchestrator Agent (`[ORCHESTRATOR-AGENT]`)
- **Responsibility:** High-level system architecture, user exploration (`/opsx-explore`), inspecting `IDEAS.md` for user notes/concepts prior to planning, creating OpenSpec changes (`/opsx-propose`), updating `ROADMAP.md`, archiving completed phases (`/opsx-archive`), and generating exact handoff prompts for the test, code, and audit agents.
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
- **Responsibility:** Performs independent quality, security, performance, and compliance checks on completed features.
- **Workflow:** Runs tests (`go test ./...`), verifies code formatting (`gofmt`), tests Docker containers (`docker compose up`), inspects security constraints, and checks adherence to `AGENTS.md` directives and OpenSpec specifications.
- **Permissions:** STRICT READ-ONLY INSPECTION. Strictly forbidden from creating, editing, refactoring, or modifying any project codebase files (`*.go`, `*_test.go`, `Dockerfile`, `docker-compose.yml`, etc.). Only outputs an audit report and the next copy-paste handoff prompt. If defects or compliance issues are found, flags them and outputs a prompt for `[CODE-AGENT]` or `[TEST-AGENT]` to resolve. If clean, approves and outputs prompt for archiving or next step.

### 4. Explicit Role Identification & Announcement Rule
- **Role Declaration:** Every AI agent invoked MUST inspect the user prompt and `AGENTS.md` to determine its active role (`[ORCHESTRATOR-AGENT]`, `[TEST-AGENT]`, `[CODE-AGENT]`, or `[AUDIT-AGENT]`).
- **First-Line Announcement:** The agent MUST announce its active role in the very first sentence of its response (e.g. `🎭 Действую в роли [ORCHESTRATOR-AGENT]` or `🧪 Действую в роли [TEST-AGENT]`).
- **Strict Boundary Enforcement:** The agent MUST NOT perform actions outside its assigned role permissions.

### 5. Guided Prompt Handoff Protocol
- **Strict Role Pause:** When an agent finishes its designated role task (e.g. `[TEST-AGENT]` finishes writing failing tests in RED state, or `[CODE-AGENT]` makes tests pass in GREEN state), it MUST NOT automatically switch roles. It MUST stop execution, commit its changes, update task status, and output a ready-to-use copy-paste prompt for the user to send to the next agent in sequence (`[ORCHESTRATOR-AGENT]` -> `[TEST-AGENT]` -> `[CODE-AGENT]` -> `[AUDIT-AGENT]`).
- **Copy-Paste Prompt Format:** Every agent handoff MUST output a formatted block containing the exact prompt the user should copy and paste for the next role or step.
- **Explicit Trigger Type Labeling:** Handoff prompts MUST explicitly state whether the prompt is:
  - ⚡ **OpenSpec Slash Command:** Start with `/opsx-explore` by default when initiating a new project phase/step (or `/opsx-propose`, `/opsx-apply`, `/opsx-archive` when managing ongoing/archiving phases).
  - 💬 **Plain Text Prompt:** Standard text prompt without slash commands for intermediate TDD roles (`[TEST-AGENT]`, `[CODE-AGENT]`, `[AUDIT-AGENT]`).
- **Exploration First Policy:** When handing off to launch a NEW project phase, the generated handoff prompt MUST ALWAYS default to starting with `/opsx-explore` so `[ORCHESTRATOR-AGENT]` inspects `IDEAS.md`, architectural trade-offs, and user concepts BEFORE generating formal specs with `/opsx-propose`.
- **OpenSpec Transition Lifecycle:**
  1. `/opsx-explore` (Orchestrator) → Discusses architecture/IDEAS.md. When aligned with user, Orchestrator outputs a handoff prompt starting with ⚡ `/opsx-propose`.
  2. `/opsx-propose` (Orchestrator) → Creates OpenSpec change artifacts (`proposal.md`, `design.md`, `specs/`, `tasks.md`), then outputs a handoff prompt for 💬 `[TEST-AGENT]`.
  3. `[TEST-AGENT]` → Writes RED failing tests, then outputs handoff prompt for 💬 `[CODE-AGENT]`.
  4. `[CODE-AGENT]` → Writes GREEN implementation, then outputs handoff prompt for 💬 `[AUDIT-AGENT]`.
  5. `[AUDIT-AGENT]` → Audits code/tests/docker. If 100% green, approves phase and outputs handoff prompt starting with ⚡ `/opsx-archive` for Orchestrator.
  6. `/opsx-archive` (Orchestrator) → Archives change to `openspec/changes/archive/`, syncs specs, updates `ROADMAP.md` checkboxes `- [x]`, and outputs handoff prompt starting with ⚡ `/opsx-explore` for the NEXT phase.

---

## Git Commit Format & Rules

Every task completion MUST be immediately committed to Git.

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
- **Error Handling:** Explicit error checking. Never swallow or suppress errors.
- **Imports:** Standard Go library imports grouped separately from third-party or internal packages.
- **Proactive Documentation Maintenance:** Whenever project architecture, workflow rules, agent permissions, or conventions evolve, AI agents MUST automatically update `AGENTS.md` and `README.md` without needing explicit reminders from the user. When completing or archiving a project phase, AI agents MUST update the corresponding checkboxes `- [x]` in `ROADMAP.md`.
- **Ideas & Backlog Inspection:** Before proposing or exploring new changes (`/opsx-explore`, `/opsx-propose`), `[ORCHESTRATOR-AGENT]` MUST inspect `IDEAS.md` to incorporate user thoughts, notes, and feature concepts into OpenSpec proposals.
- **Public Repository & Security Hygiene:** SimpleCloud may become a public repository. ALL AI agents MUST strictly prevent committing sensitive credentials, secrets, private keys, API tokens, or hardcoded passwords into Git. Environment variables (`.env`) MUST be used for configuration. Personal scratchpad notes in `IDEAS.md` are kept local via `.gitignore`.
