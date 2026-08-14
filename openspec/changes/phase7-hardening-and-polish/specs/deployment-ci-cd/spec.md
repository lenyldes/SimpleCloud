# deployment-ci-cd delta — phase7-hardening-and-polish

## ADDED Requirements

### Requirement: Fail-Fast Pipeline Ordering
The CI/CD workflow SHALL NOT execute any downstream job when an upstream job failed or was cancelled: deployment SHALL require successful lint and test jobs, and jobs that do not declare explicit dependencies SHALL be gated so a failed earlier stage prevents later stages from running.

#### Scenario: Lint failure blocks tests and deploy
- **WHEN** the lint job fails on a push to `main`
- **THEN** the test and deploy jobs SHALL NOT run (skipped or never started), and no deployment SHALL occur.

#### Scenario: Test failure blocks deploy
- **WHEN** the lint job passes but the test job fails
- **THEN** the deploy job SHALL NOT run and the production server SHALL not be touched.

### Requirement: Deterministic Dependency Build Layer
The storage service Docker build SHALL copy both `go.mod` and `go.sum` into the build stage before running `go mod download`, so module downloads are verified against recorded checksums and the dependency layer is cache-deterministic.

#### Scenario: Docker build downloads verified dependencies
- **WHEN** the storage service image is built
- **THEN** the build SHALL execute `go mod download` only after both `go.mod` and `go.sum` are present in the build context working directory, and the build SHALL succeed with verified module checksums.
