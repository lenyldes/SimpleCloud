## ADDED Requirements

### Requirement: Modular CSS Architecture and In-Place Style Index
The web frontend SHALL structure its styling rules into dedicated modular stylesheets under `css/` (`base.css`, `layout.css`, `components.css`, `modals.css`), loaded concurrently via separate `<link rel="stylesheet">` elements in `index.html`, and SHALL maintain an in-place documentation index in `css/README.md` and `docs/DESIGN_SPEC.md` mapping all selectors and UI components to their responsible modules to enable zero-scanning navigation.

#### Scenario: Browser loads modular stylesheets concurrently
- **WHEN** a user requests the web application root
- **THEN** `index.html` SHALL link `css/base.css`, `css/layout.css`, `css/components.css`, and `css/modals.css` in cascade order without referencing a monolithic `styles.css`.

#### Scenario: Complete selector and visual design preservation
- **WHEN** the modular stylesheets are applied to the DOM
- **THEN** all design tokens, typography, layout dimensions, transitions, and all 73 UI component classes SHALL render with 100% visual parity to the monolithic baseline.

#### Scenario: In-place style index navigation
- **WHEN** an engineer or agent inspects `services/web-frontend/src/css/README.md`
- **THEN** the index SHALL specify the exact module location for every major UI element, including buttons, layout grids, tables, dropdowns, modals, and overlays.
