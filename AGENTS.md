# SimpleCloud - Agent Directives & Architecture Guide

## Project Overview
SimpleCloud is a lightweight, super-fast, self-hosted cloud storage web application built with Go, PostgreSQL, Docker Compose, Caddy reverse proxy, and Vanilla HTML/JS.

---

## Agent Roles & TDD Rules

All AI agents working on this codebase MUST strictly adhere to Test-Driven Development (TDD) role separation:

### 1. Test Agent (`[TEST-AGENT]`)
- **Responsibility:** Writes and updates unit & integration tests (`*_test.go`) based on OpenSpec requirements.
- **Workflow:** Writes failing tests first (**RED** state) before any feature implementation code is written.
- **Permissions:** Only writes and modifies test files (`*_test.go`). Must NOT modify production implementation code (`*.go` files outside of tests).

### 2. Code Implementation Agent (`[CODE-AGENT]`)
- **Responsibility:** Writes and refactors production logic (`*.go`) to make failing tests pass (**GREEN** state).
- **Permissions:** Strictly FORBIDDEN from editing, altering, disabling, commenting out, or deleting any `*_test.go` files.
- **Protocol on Test Issues:** If a test appears invalid or buggy, `[CODE-AGENT]` MUST NOT fix the test itself. It must pause and request `[TEST-AGENT]` to review and adjust the test.

### 3. Guided Prompt Handoff Protocol
- **Strict Role Pause:** When an agent finishes its designated role task (e.g. `[TEST-AGENT]` finishes writing failing tests in RED state), it MUST NOT automatically switch roles and write production code. It MUST stop execution, commit its changes, update task status, and output a ready-to-use copy-paste prompt for the user to send to the next agent (`[CODE-AGENT]`).
- **Copy-Paste Prompt Format:** Every agent handoff MUST output a formatted block containing the exact prompt the user should copy and paste for the next role or step.


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


