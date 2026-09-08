# SimpleCloud - Modular CSS Architecture & Navigation Guide

## Overview

SimpleCloud uses a zero-build, ultra-fast modular CSS architecture. To keep CSS maintainable and prevent monolithic files exceeding AI agent token windows, styles are split into 4 cohesive modules located in `services/web-frontend/src/css/`.

Each module is strictly capped at **<= 450 lines** to ensure clean readability, instant comprehension, and zero truncation during automated inspection.

---

## Modular Stylesheets & Cascade Loading Order

Stylesheets are linked concurrently via `<link rel="stylesheet">` tags in `services/web-frontend/src/index.html` in strict cascade order:

1. **`base.css`**: Design tokens (`:root`), typography, `@font-face` (Inter), global `*` box-sizing/reset, `body`, accessibility and visibility utilities (`.visually-hidden`, `.hidden`).
2. **`layout.css`**: Global page layout and macro-structural containers (`#app`, `#app-header`, `.brand-*`, `.search-*`, `#search-input`, `.header-actions`, `.app-body`, `#app-sidebar`, `.nav-*`, `.main-content`, `.toolbar-*`, `#breadcrumbs-bar`, `.sort-select`, `.view-toggle`).
3. **`components.css`**: Reusable UI components, widgets, and controls (`.btn*`, `.user-profile*`, `.avatar`, `.profile-dropdown*`, `#quota-container*`, `#workspace`, `.file-grid`, `.grid-card*`, `.file-list*`, `.empty-state*`, `#toast-container`, `.toast*`).
4. **`modals.css`**: Dialogs, interactive popups, fullscreen overlays, and embedded media viewers (`#dropzone-overlay*`, `.modal-backdrop`, `.modal-dialog`, `.modal-header/body/footer`, `#modal-auth*`, `.lightbox-*`, `.code-pre`, `.video-player`, `.form-*`).

---

## Component-to-File Lookup Table

Use this lookup table for instant navigation without searching across multiple files:

| Component / UI Element | Selectors / Classes / IDs | Module File |
| :--- | :--- | :--- |
| **Design Tokens & Variables** | `:root`, `--color-*`, `--font-*`, `--shadow-*` | `base.css` |
| **Font & Global Reset** | `@font-face`, `*`, `body` | `base.css` |
| **Utility Classes** | `.hidden`, `.visually-hidden` | `base.css` |
| **Header Bar & Logo** | `#app-header`, `.brand-container`, `.brand-icon`, `.brand-title` | `layout.css` |
| **Search Input Bar** | `.search-container`, `.search-icon`, `#search-input` | `layout.css` |
| **Navigation Sidebar** | `.app-body`, `#app-sidebar`, `.nav-menu`, `.nav-item`, `.nav-icon` | `layout.css` |
| **Breadcrumbs & Toolbar** | `.toolbar-container`, `#breadcrumbs-bar`, `.breadcrumb-*` | `layout.css` |
| **View Mode & Sort Controls** | `.toolbar-controls`, `.sort-select`, `.view-toggle` | `layout.css` |
| **Buttons & Icons** | `.btn`, `.btn-primary`, `.btn-secondary`, `.btn-outline`, `.btn-icon` | `components.css` |
| **User Profile & Dropdown** | `.user-profile`, `.avatar`, `.profile-chevron`, `.profile-dropdown*`, `.btn-logout` | `components.css` |
| **Quota Usage Widget** | `#quota-container`, `.quota-header`, `.quota-progress-*`, `.quota-warning`, `.quota-danger`, `.quota-text` | `components.css` |
| **Workspace & Container** | `#workspace` | `components.css` |
| **File Grid View** | `.file-grid`, `.grid-card`, `.grid-card-icon`, `.grid-card-name`, `.grid-card-meta`, `.card-actions` | `components.css` |
| **File List Table View** | `.file-list`, `.file-list th`, `.file-list td`, `.file-list tr`, `.list-name-col`, `.list-icon` | `components.css` |
| **Empty State Placeholder** | `.empty-state`, `.empty-icon`, `.empty-title` | `components.css` |
| **Toast Notifications** | `#toast-container`, `.toast`, `.toast-success`, `.toast-danger` | `components.css` |
| **Drag & Drop Upload Zone** | `#dropzone-overlay`, `.dropzone-box`, `.dropzone-icon`, `.dropzone-title` | `modals.css` |
| **Modal Dialog Scaffolding** | `.modal-backdrop`, `.modal-dialog`, `.modal-header`, `.modal-title`, `.modal-body`, `.modal-footer` | `modals.css` |
| **Authentication Modal** | `#modal-auth`, `.auth-error-msg` | `modals.css` |
| **Image Lightbox Viewer** | `.lightbox-dialog`, `.lightbox-img`, `.lightbox-actions` | `modals.css` |
| **Code & Text File Viewer** | `.code-pre` | `modals.css` |
| **Video Player Viewer** | `.video-player` | `modals.css` |
| **Form Inputs & Groups** | `.form-group`, `.form-label`, `.form-input` | `modals.css` |

---

## Development Guidelines for Agents

- **Zero Monoliths**: Never consolidate styles into a single file or exceed 450 lines per module.
- **Strict Cascade**: New global rules belong in `base.css`, structural containers in `layout.css`, atomic components in `components.css`, and overlays/dialogs in `modals.css`.
- **Zero Build Step**: Stylesheets are pure Vanilla CSS served directly via Nginx.
