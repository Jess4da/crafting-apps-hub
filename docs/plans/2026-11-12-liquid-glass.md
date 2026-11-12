# Liquid Glass Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the paper/ink editorial UI with a clear Liquid Glass look over the OS window material, with an opaque fallback where no OS glass exists.

**Architecture:** Go decides once at startup whether the OS can provide glass (`osGlass()`), configures the window material, and passes `?glass=1` in the window URL. A pre-paint script turns that into `html.os-glass`. All visuals come from a new Skeleton theme plus glass tokens in `layout.css`; components consume tokens only.

**Tech Stack:** Go 1.25, Wails v3 beta.28, SvelteKit 3, Svelte 5 runes, Skeleton 5, Tailwind 4, bun.

**Spec:** `docs/specs/2026-11-12-liquid-glass-design.md`

## Global Constraints
- No new npm or Go dependencies. Remove `@fontsource-variable/fraunces`, `@fontsource/ibm-plex-sans`, `@fontsource/ibm-plex-mono`.
- Components use tokens only: no raw hex/oklch/rgb outside `glass.css` and `layout.css`.
- Never nest `backdrop-filter`. Catalogue rows are content, not glass.
- Light/dark follows `prefers-color-scheme`; no manual toggle, no `localStorage.mode`.
- `--font-mono` only for versions, counts, paths.
- Button labels never wrap; no horizontal scroll at 320/375/414/768 px.
- `frontend/bindings/` untouched (no binding changes in this plan).
- `cd frontend && bun run check` ends with 0 errors and 0 warnings.

## Review Focus
- Windows with "Transparency effects: Off": window must paint the fallback surface, not black. Task 2's `layout.css` media rule; checked in Task 4 smoke.
- Windows 10 / build < 22621: `osGlass()` false, so the window is solid and has no `?glass=1`. Task 1's `TestAcrylicBuild`.
- Server mode in a browser: never adds `os-glass`. `osGlass()` returns false under `serverMode`, which Task 1 covers.
- Dark desktop behind light OS theme: muted 13 px text on the scrim must still reach AA (4.5:1). Task 4 smoke, measured with devtools contrast against `#000` and `#fff` page backgrounds.
- Startup: no flash of the fallback surface on OS-glass launches. `app.html` runs the script before paint (Task 2); checked in Task 4.

---

### Task 1: OS glass detection and window material

**Files:**
- Create: `glass_windows.go`, `glass_darwin.go`, `glass_linux.go`, `glass_test.go`
- Modify: `main.go:58-71`

**Interfaces:**
- Produces: `func osGlass() bool` (package main); window URL `/?glass=1` when true, `/` otherwise.

- [ ] **Step 1: Write the failing test** in `glass_test.go` (`//go:build windows`): `TestAcrylicBuild` asserting `acrylicBuild(22621)` and `acrylicBuild(26100)` are true; `acrylicBuild(22000)`, `acrylicBuild(19045)`, `acrylicBuild(17763)` are false. Signature: `func acrylicBuild(build uint32) bool` in `glass_windows.go`.
- [ ] **Step 2: Run** `go test -run TestAcrylicBuild .` on Windows. Expected: FAIL, `undefined: acrylicBuild`.
- [ ] **Step 3: Implement**
  - `glass_windows.go`: `acrylicBuild` above; `osGlass()` returns `!serverMode && acrylicBuild(windows.RtlGetVersion().BuildNumber)` (`golang.org/x/sys/windows`).
  - `glass_darwin.go`: `osGlass()` returns `!serverMode`.
  - `glass_linux.go`: `osGlass()` returns `false`.
  - `main.go`: `glass := osGlass()`; set `URL` to `"/?glass=1"` when `glass`, else `"/"`; when `glass`, set `BackgroundType: application.BackgroundTypeTranslucent`, `Windows: application.WindowsWindow{BackdropType: application.Acrylic, Theme: application.SystemDefault}`, `BackgroundColour: application.NewRGBA(0, 0, 0, 0)`; otherwise keep `BackgroundColour` `NewRGB(18, 18, 22)`. Mac options unchanged.
- [ ] **Step 4: Run** `go test -run TestAcrylicBuild .`, `go vet ./...`, `go build -tags server -o bin/hub-server .`. Expected: PASS, no vet output, build succeeds.
- [ ] **Step 5: Commit** `git add glass_*.go main.go && git commit -m "feat: detect OS glass and enable Acrylic window material"`

### Task 2: Glass theme and tokens

**Files:**
- Create: `frontend/src/routes/glass.css`
- Delete: `frontend/src/routes/atelier.css`
- Modify: `frontend/src/routes/layout.css`, `frontend/src/app.html`, `frontend/src/routes/+layout.svelte` (toast styles), `frontend/package.json`, `frontend/bun.lock`

**Interfaces:**
- Consumes: `?glass=1` from Task 1.
- Produces (used by Task 3; light/dark values per spec):
  - classes on `<html>`: `data-theme="glass"`, `os-glass` (only when OS glass is present)
  - tokens: `--glass-fill`, `--glass-fill-strong`, `--glass-hover`, `--glass-rim`, `--glass-shadow`, `--glass-inner`, `--glass-scrim`, `--glass-blur`, `--surface-fallback`, `--color-ink`, `--color-muted`, `--color-rule`, `--color-accent`, `--color-accent-fill`, `--color-accent-ink`, `--color-accent-hover`, `--color-danger`, `--color-danger-fill`, `--color-danger-ink`, `--color-warn-text`, `--color-focus`, `--color-progress`, `--color-backdrop`, `--font-body`, `--font-mono`, `--ease-snappy` `cubic-bezier(0.34, 1.56, 0.64, 1)`, `--ease-out`, `--dur-micro`, `--dur-short`
  - utility class `.glass` (fill + rim + shadow + inner; no blur) and `.glass--float` (`--glass-fill-strong` + `backdrop-filter: blur(var(--glass-blur))`, for menus/dialogs/toasts)
  - restyled shared classes: `.act` (+`--accent`, `--danger`, `--quiet`), `.more`, `.menu`
  - `.page` background rule: `--glass-scrim` under `html.os-glass`, else transparent
- Removed: `--color-paper`, `--color-paper-2`, `--color-rule-2`, `--color-shadow`, `--font-display`

- [ ] **Step 1: Write `glass.css`** as a Skeleton 5 theme (same variable set as the deleted `atelier.css`): neutral surface ramp hue 250 chroma ≤ 0.01; primary ramp centred on `#0088ff` at 500; error/warning/success ramps; `--typo-*` font families `var(--font-body)`; heading weight 600, letter-spacing −0.01em; `--radius-base` 0.75rem; `--radius-container` 1.25rem.
- [ ] **Step 2: Rewrite `layout.css`**: drop `@fontsource*` imports and `atelier.css`, import `glass.css`; dark variant becomes `@custom-variant dark (@media (prefers-color-scheme: dark));`; light tokens in `:root`, dark in `@media (prefers-color-scheme: dark) { :root { … } }`; `body { background: var(--surface-fallback) }`, and `html.os-glass body { background: transparent }` unless `@media (prefers-reduced-transparency: reduce)`, which keeps `--surface-fallback`, makes `--glass-fill*`/`--glass-scrim` opaque and sets `--glass-blur: 0px`; `.page` rule as in Interfaces; `.act`/`.more`/`.menu` per spec "Buttons" and "Shapes/motion"; keep the reduced-motion block.
- [ ] **Step 3: `app.html`**: `data-theme="glass"`; the pre-paint script becomes `if (new URLSearchParams(location.search).has("glass")) document.documentElement.classList.add("os-glass");`.
- [ ] **Step 4: `+layout.svelte`**: `.note` uses `.glass--float` styling (radius 18px, danger rim via `box-shadow` ring, not a 1 px border).
- [ ] **Step 5: Remove fonts:** `cd frontend && bun remove @fontsource-variable/fraunces @fontsource/ibm-plex-sans @fontsource/ibm-plex-mono`.
- [ ] **Step 6: Verify:** `cd frontend && bun run check` and `grep -rnE "atelier|fontsource|--color-paper|--font-display" frontend/src` (Task 3 files excluded until Task 3 lands). Expected: check passes for these files; grep hits only `+page.svelte` / `AppCard.svelte`.
- [ ] **Step 7: Commit** `git add frontend/src/routes/glass.css frontend/src/routes/layout.css frontend/src/app.html frontend/src/routes/+layout.svelte frontend/package.json frontend/bun.lock && git rm frontend/src/routes/atelier.css && git commit -m "feat(ui): liquid glass theme and tokens"`

### Task 3: Glass components

**Files:**
- Modify: `frontend/src/routes/+page.svelte`, `frontend/src/lib/AppCard.svelte`

**Interfaces:**
- Consumes: every token/class in Task 2's Produces list. Must not reference any token in its Removed list.

- [ ] **Step 1: Remove the mode toggle:** delete `dark` state, `toggleMode`, the `rail__mode` `<li>` and its CSS, and the `onMount` line that reads `.dark`.
- [ ] **Step 2: Rail:** `.rail` becomes a `.glass` panel, radius 22px, inset 8px from window edges (sticky height `calc(100dvh - 16px)`); active filter = `--glass-hover` pill fill with ink text (no underline); counts keep `--font-mono`; `.rail__count[data-accent]` keeps `--color-accent`. Below 48rem: the strip is one horizontal `.glass` pill bar, scrollable, no wrapping.
- [ ] **Step 3: Header/footer:** headings use `--font-body` weight 600 (title `clamp(1.75rem, 2vw + 1rem, 2.5rem)`); `.search` becomes a `.glass` pill; `.foot` sits in a `.glass` bar, radius 18px; `.path` uses `--glass-fill` instead of `--color-paper-2`.
- [ ] **Step 4: Sheets:** `.sheet` uses `.glass--float`, radius 28px; `.sheet__backdrop` uses `--color-backdrop` with no blur (sheet already blurs; never nest).
- [ ] **Step 5: AppCard:** `.row__name` drops `--font-display`/`font-variation-settings`, uses `--font-body` 600 1.125rem (1rem below 40rem); row separators stay `--color-rule` hairlines; icon radius 0.875rem; progress track `--glass-fill`, range `--color-progress`; ghost rows in `+page.svelte` match.
- [ ] **Step 6: Verify:** `cd frontend && bun run check` (0 errors, 0 warnings) and `grep -rnE "--color-paper|--color-rule-2|--color-shadow|--font-display|localStorage" frontend/src` (no output).
- [ ] **Step 7: Commit** `git add frontend/src/routes/+page.svelte frontend/src/lib/AppCard.svelte && git commit -m "feat(ui): glass rail, header, footer, sheets and rows"`

### Task 4: Docs and visual verification

**Files:**
- Modify: `AGENTS.md` (Key Directories `atelier.css` row; "Design system" bullet and its sub-bullets; frontend `layout.css` description)

**Interfaces:**
- Consumes: Tasks 1–3 merged.

- [ ] **Step 1: Rewrite AGENTS.md design rules** to describe: Liquid Glass (clear), OS material via `osGlass()` + `?glass=1` + `html.os-glass`, fallback surface rules (no OS glass / reduced transparency), the glass token list, system fonts, OS-driven light/dark, never-nest-glass, rows are content. Keep the "one primary action + ⋯ menu", "success is silent", label-wrap and 320/375/414/768 rules. Replace the `atelier.css` row with `glass.css`, mention the new `glass_*.go` files under Architecture.
- [ ] **Step 2: Server-mode check:** `cd frontend && bun run build && cd .. && go build -a -tags server -o bin/hub-server . && WAILS_SERVER_PORT=8765 ./bin/hub-server`; in a browser at 320/375/414/768/1180 px, light and dark (devtools emulation): fallback surface painted, no horizontal scroll, labels unwrapped, menu and both dialogs open, `html` has no `os-glass`. Then open `/?glass=1` with a `#000` and a `#fff` page background (devtools) and confirm muted text ≥ 4.5:1 on the scrim in both schemes.
- [ ] **Step 3: Desktop check (Windows 11):** `wails3 build`, copy the exe to a temp folder, run it: desktop visible through Acrylic, no fallback flash on launch; toggle Settings → Personalization → Colors → Transparency effects Off and confirm the window repaints with the fallback surface (not black); switch OS theme and confirm light/dark follows.
- [ ] **Step 4: Commit** `git add AGENTS.md && git commit -m "docs: describe liquid glass design system"`
