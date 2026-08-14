## ADDED Requirements

### Requirement: Self-Hosted Typography Assets
The web frontend SHALL serve its typography (the Inter font family) from its own static assets instead of external CDNs, so the page renders with the intended design under the existing Content-Security-Policy (`style-src 'self'`, `default-src 'self'`) without any CSP violation and without third-party requests.

#### Scenario: No blocked font or style resources
- **WHEN** a user loads the application in a browser
- **THEN** the browser console SHALL contain no CSP violations for styles or fonts, and no requests to external font CDNs SHALL be made.

#### Scenario: Inter font served from own origin
- **WHEN** the page renders
- **THEN** the Inter font files SHALL be fetched from the frontend's own origin via `@font-face` rules referencing local woff2 assets.

#### Scenario: External Google Fonts links removed
- **WHEN** the page source is inspected
- **THEN** there SHALL be no `preconnect` or stylesheet `<link>` elements referencing `fonts.googleapis.com` or `fonts.gstatic.com`.
