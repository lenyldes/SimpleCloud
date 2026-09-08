## ADDED Requirements

### Requirement: Universal File Line Limit Architecture Guard
The project SHALL enforce an automated architecture guard ensuring that every readable text-based or human-readable file across the repository does not exceed 450 lines of code or text. This constraint SHALL apply by universal principle to all readable file formats (including `.go`, `.js`, `.css`, `.html`, `.md`, `.sql`, `.sh`, `.yml`, `.yaml`, `.conf`, Dockerfile, and arbitrary future text extensions) and SHALL exclude only strictly non-text binary files (detected via known binary extensions such as `.woff2` and images, or presence of null bytes `0x00` and invalid UTF-8) and `.git/` metadata.

#### Scenario: Architecture test enforces file length ceiling
- **WHEN** the automated test suite executes `TestMaxFileLineCount`
- **THEN** every tracked text file in the repository SHALL contain 450 or fewer lines of text, and any file exceeding 450 lines SHALL cause the test suite to fail with the file path and exact line count.

#### Scenario: Binary files and git directory are excluded from line count guard
- **WHEN** the architecture guard inspects tracked repository assets
- **THEN** binary files containing null bytes `0x00` or known binary extensions (e.g. `.woff2`) and `.git/` metadata SHALL be skipped without triggering false failures.
