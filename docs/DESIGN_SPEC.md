# SimpleCloud - Web UI Design Specification & Design Tokens

> **Reference Source:** Analyzed and extracted from Mail.ru Cloud Web UI reference (`design_ref/Все файлы _ Облако Mail.html`).
> **Goal:** High-performance, ultra-clean, modern Vanilla HTML/CSS/JS frontend modeled after Mail.ru Cloud aesthetics.

---

## 🎨 Design System & CSS Custom Properties

All styling in `services/web-frontend/src/styles.css` must use these core CSS custom properties defined on `:root`:

```css
:root {
  /* Brand & Accent Colors (Cloud Mail.ru Blue) */
  --color-primary: #0077ff;
  --color-primary-hover: #005fd1;
  --color-primary-light: #e5f2ff;
  --color-primary-subtle: #f0f7ff;

  /* Neutral Backgrounds & Surfaces */
  --color-bg-app: #f5f5f7;
  --color-bg-surface: #ffffff;
  --color-bg-secondary: #f0f1f3;
  --color-bg-hover: #ebedf0;
  
  /* Text & Content Colors */
  --color-text-main: #2c2d2e;
  --color-text-secondary: #818c99;
  --color-text-muted: #99a2ad;
  --color-text-inverse: #ffffff;

  /* Borders & Dividers */
  --color-border: #ebedf0;
  --color-border-dark: #d7d8db;

  /* Functional Status Colors */
  --color-danger: #e64646;
  --color-danger-light: #ffeaea;
  --color-warning: #ff9e00;
  --color-success: #4bb34b;

  /* Typography */
  --font-family-base: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  --font-family-accent: 'VK Sans Display', 'MailSans', 'Inter', -apple-system, sans-serif;

  /* Font Sizes */
  --font-size-xs: 12px;
  --font-size-sm: 13px;
  --font-size-md: 14px;
  --font-size-lg: 16px;
  --font-size-xl: 18px;
  --font-size-title: 22px;

  /* Layout Dimensions */
  --header-height: 60px;
  --sidebar-width: 240px;
  --border-radius-sm: 6px;
  --border-radius-md: 10px;
  --border-radius-lg: 16px;
  --border-radius-pill: 999px;

  /* Shadows & Depth */
  --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.08);
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.12);

  /* Transitions */
  --transition-fast: 0.15s ease;
  --transition-normal: 0.25s ease;
}
```

---

## 📐 Layout Architecture

The application layout consists of 3 main regions:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. HEADER (Logo | Search Bar | Action Buttons | User Avatar)                │
├───────────────┬─────────────────────────────────────────────────────────────┤
│ 2. SIDEBAR    │ 3. MAIN WORKSPACE                                           │
│               │                                                             │
│ - Navigation  │ - Breadcrumbs & Toolbar (Grid/List toggle, Sorting)         │
│   List        │ - Drag-and-Drop Dropzone Overlay                            │
│ - Quota Meter │ - File & Folder Content Area (Grid cards / List rows)       │
│               │                                                             │
└───────────────┴─────────────────────────────────────────────────────────────┘
```

### 1. Header Bar (`--header-height: 60px`)
- **Left**: `SimpleCloud` Brand Logo (Cloud SVG icon in `--color-primary` + bold title).
- **Center**: Search input bar (rounded `--border-radius-md`, background `--color-bg-hover`, placeholder "Search files...").
- **Right**:
  - `+ Upload` button (Filled pill button `--color-primary`, white text).
  - `+ New Folder` button (Outline button with icon).
  - User profile avatar & dropdown menu.

### 2. Left Sidebar (`--sidebar-width: 240px`)
- Vertical navigation menu:
  - 📂 **All Files** (`/`)
  - ⭐ **Favorites**
  - 🗑️ **Trash**
- Active item styling: background `--color-primary-light`, text `--color-primary`, font-weight `600`.
- Bottom Quota Card:
  - Text: `Used 2.4 GB of 5 GB`
  - Progress bar track (`#EBEDF0`) with blue fill (`#0077FF`). Turns amber/red when >85% capacity.

### 3. Workspace & File Grid
- **Breadcrumbs Bar**: Folder path hierarchy navigation (`All Files / Projects / 2026`).
- **Toolbar**: View Switcher (Grid icon ⏹️ vs List icon ☰), Sort selector (Name, Size, Modified Date).
- **File & Folder Items**:
  - **Folders**: Distinct blue folder icons (`#0077FF`), item count subtitle.
  - **Files**: MIME-specific SVG icons (Image 🖼️, Video 🎬, Code/Text 📝, Archive 📦, PDF 📄).
  - **Hover state**: Light background highlight `--color-bg-hover`, shadow `--shadow-sm`, context action buttons (Download 📥, Delete 🗑️).
- **Drag & Drop Overlay**:
  - When dragging files into window, a full-screen or section overlay appears with `--color-primary-light` background, blue dashed border, and text "Drop files here to upload".

---

## 👁️ Preview & Viewer Modals

1. **Image Viewer (Lightbox)**: Fullscreen modal with black overlay `rgba(0, 0, 0, 0.75)`, centered image, download and close buttons.
2. **Text / Code Viewer**: Clean modal box with monospaced font viewer, line numbers, and download button.
3. **Video Player**: HTML5 `<video controls>` player centered overlay.

---

## 🛠️ Implementation Rules for Code Agents

When `[CODE-AGENT]` builds the `services/web-frontend` package:
1. Put all CSS rules in `services/web-frontend/src/styles.css` using the `:root` design tokens.
2. Do NOT import third-party CSS libraries (Bootstrap, Tailwind, etc.) — strictly Vanilla CSS adhering to `docs/DESIGN_SPEC.md`.
3. Use SVG icons for folders, file types, and buttons matching the blue Mail.ru Cloud color palette.
