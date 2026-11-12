# Liquid Glass redesign

## Intent
Replace the editorial "Index-First" paper/ink look with a Liquid Glass look (clear variant): the hub window itself is see-through/frosted by the OS, and chrome floats on it as glass. Same stack: Skeleton 5 theme + Tailwind 4 + Lucide. No new dependencies.

## Decided by the user
- **Replace** the current look entirely (not a third theme). AGENTS.md design rules are rewritten to match.
- **Backdrop = OS window material**: Windows 11 Acrylic (build ≥ 22621), macOS vibrancy (already `MacBackdropTranslucent`). Everything else (Linux, Windows 10, server mode in a browser) has no OS glass and paints an opaque, tokenised fallback surface. Wails' legacy Windows 10 blur-behind is not used.
- **Reduced transparency** (`prefers-reduced-transparency: reduce`; on Windows this is "Transparency effects: Off", which also disables Acrylic): the whole window paints the opaque fallback surface, same as no OS glass. Glass shapes/rims stay; nothing is see-through.
- **Contrast**: WCAG AA against any desktop. The content column carries a faint scrim tinted like the OS material (about +15 % opacity); chrome stays clear.
- **Clear transparency**: thin, low-tint fills so the desktop shows through.

## Goals
- Window shows the desktop through OS blur on Windows 11 and macOS.
- Chrome (rail, header tools, footer, menus, dialogs, toasts) is glass: translucent fill, light rim, soft shadow, inner highlight.
- Text stays WCAG AA over any desktop wallpaper, light or dark, thanks to the content scrim; checked against pure white and pure black desktops in both OS themes.
- Respects `prefers-reduced-transparency` (whole window opaque fallback) and `prefers-reduced-motion`.
- Existing behaviour unchanged: one primary action per row, `⋯` menu sections, Update all, failure-only toasts, loading ghosts, 320–768 px responsive rules.

## Non-goals / anti-goals
- No iOS component clones (tab bars, large-title nav). Desktop launcher layout stays: rail + catalogue list.
- No `@ios27_design_system/*` packages, no new npm/Go deps.
- Never nest glass (no `backdrop-filter` inside a `backdrop-filter` surface). Rows are content, not glass.
- No raw colours in components: tokens only.
- Failure: unreadable text on a bright wallpaper; foggy stacked blur; horizontal scroll at 320 px.

## The how (decided)
- **Window** (`main.go`): when `osGlass()` is true, Windows gets `BackgroundType: BackgroundTypeTranslucent`, `BackdropType: Acrylic`, and the window `URL` becomes `/?glass=1`; otherwise the window stays solid and `URL` is `/`. `BackgroundColour` alpha 0 when translucent. `Theme: SystemDefault`. Mac options unchanged.
- **OS glass flag** (`osGlass() bool`, build-tagged): Windows = build ≥ 22621 via `windows.RtlGetVersion()` (`golang.org/x/sys`, already a dependency); darwin = true; linux = false; always false when `serverMode`. No binding: `app.html`'s pre-paint script adds `html.os-glass` when `location.search` has `glass=1`, so there is no startup flash. *(my call: avoids the flash an async binding would cause)*
- **Fallback surface**: `body` paints `--surface-fallback` unless `html.os-glass` **and** not `prefers-reduced-transparency: reduce`; in that case `body` is transparent and `.page` (the content column) paints `--glass-scrim`.
- **Light/dark follows the OS** (`prefers-color-scheme`), because the Acrylic/vibrancy tint follows the OS theme; a manual dark CSS over a light OS material gives grey-on-grey. The "Ink mode" toggle and `localStorage.mode` are removed. *(my call)*
- **Theme**: new Skeleton theme `routes/glass.css` (`data-theme="glass"`) replacing `atelier.css`: neutral surface ramp, primary = system blue `#0088ff` ramp, error/warning/success ramps.
- **Glass tokens** in `layout.css` (light / dark):
  - `--glass-fill`: white 0.18 / black 0.22 (clear; rail, header, footer)
  - `--glass-fill-strong`: white 0.70 / `#1a1a1a` 0.70 (menus, dialogs, toasts: must read over arbitrary content)
  - `--glass-hover`: white 0.28 / white 0.08
  - `--glass-rim`: `#dbdbdb` / `#a6a6a6` as a 0.5 px ring (`box-shadow: 0 0 0 0.5px`)
  - `--glass-scrim`: white 0.32 / black 0.36 (content column over OS glass; tune for AA, never above 0.5)
  - `--glass-blur`: 15px (menus/dialogs only, since in-page blur over the OS material adds fog)
  - `--glass-inner`: inset top highlight white 0.5 / 0.12
  - `--surface-fallback`: soft neutral gradient (light/dark), used when there is no OS glass or transparency is reduced
  - Keep names `--color-ink`, `--color-muted`, `--color-rule`, `--color-accent*`, `--color-danger*`, `--color-warn-text`, `--color-focus`, `--color-progress`, `--color-backdrop`; delete `--color-paper`, `--color-paper-2`, `--color-rule-2`, `--color-shadow` and all `--font-display` uses.
  - `@media (prefers-reduced-transparency: reduce)`: fills become opaque, `--glass-blur: 0px`.
- **Fonts**: `--font-body: system-ui, -apple-system, "Segoe UI Variable Text", "Segoe UI", sans-serif`; `--font-mono: ui-monospace, "SF Mono", "Cascadia Mono", Consolas, monospace` (still only for versions, counts, paths). Headings use body face, semibold. Remove the three `@fontsource*` packages.
- **Shapes/motion**: pills for buttons and search; rail panel radius 22 px inset 8 px from window edges; menus 18 px; dialogs 28 px; toasts 18 px. Snappy easing `cubic-bezier(0.34, 1.56, 0.64, 1)` for press/hover scale; menus/dialogs 250 ms stiff (no bounce).
- **Buttons**: `.act` = glass pill (`--glass-fill-strong`, rim, ink text); `.act--accent` solid system blue + white; `.act--danger` solid danger fill; `.act--quiet` no fill. Hover lightens (`--glass-hover`), no colour inversion.

## Left to the builder
Exact spacing, which header/footer elements sit inside the glass bar, ghost placeholder styling, per-breakpoint rail strip treatment, as long as the goals and the 320/375/414/768 px checks hold.
