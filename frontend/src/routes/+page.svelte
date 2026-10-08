<script lang="ts">
  /* Hallmark · genre: editorial · macrostructure: Index-First · nav: N9 rail · footer: Ft2 inline
   * Styles here reference tokens from routes/layout.css + routes/atelier.css only. */
  import { onMount } from "svelte";
  import { Dialog, Portal } from "@skeletonlabs/skeleton-svelte";
  import { Browser } from "@wailsio/runtime";
  import SearchIcon from "@lucide/svelte/icons/search";
  import type { AppState } from "#bindings/craftinghub/internal/hub/models.ts";
  import AppCard from "#lib/AppCard.svelte";
  import { hub } from "#lib/hub.svelte.ts";

  type View = "all" | "installed" | "updates" | "settings";
  let view = $state<View>("all");
  let query = $state("");
  let dark = $state(false);
  let uninstallApp = $state<AppState | null>(null);
  let cleanUninstall = $state(false);

  const shown = $derived(
    (view === "installed" ? hub.installed : view === "updates" ? hub.updates : hub.apps).filter((a) =>
      `${a.name} ${a.group} ${a.category} ${a.description}`.toLowerCase().includes(query.trim().toLowerCase()),
    ),
  );
  // Catalog order keeps each group contiguous, so a Map preserves section order.
  // Not Map.groupBy: older macOS/Linux WebKit webviews lack it.
  const groups = $derived.by(() => {
    const m = new Map<string, AppState[]>();
    for (const a of shown) m.set(a.group, [...(m.get(a.group) ?? []), a]);
    return m;
  });

  const nav = $derived([
    { id: "all" as const, label: "All apps", count: hub.apps.length },
    { id: "installed" as const, label: "Installed", count: hub.installed.length },
    { id: "updates" as const, label: "Updates", count: hub.updates.length },
  ]);

  // The index header: a count line, not a slogan.
  const heading = $derived(
    view === "installed"
      ? "On this computer"
      : view === "updates"
        ? "Updates"
        : "The Craft apps",
  );
  const summary = $derived.by(() => {
    const n = hub.apps.length;
    const i = hub.installed.length;
    const u = hub.updates.length;
    if (!n) return "";
    return [`${n} apps`, `${i} installed`, u ? `${u} with updates` : "all current"].join(" · ");
  });

  onMount(() => {
    dark = document.documentElement.classList.contains("dark");
    hub.init();
  });

  function toggleMode() {
    dark = !dark;
    document.documentElement.classList.toggle("dark", dark);
    localStorage.setItem("mode", dark ? "dark" : "light");
  }
</script>

<div class="shell">
  <nav class="rail" aria-label="Library">
    <p class="wordmark">Craft<span>Hub</span></p>
    <div class="rail__strip">
    <ul class="rail__list">
      {#each nav as item (item.id)}
        <li>
          <button
            type="button"
            class="rail__link"
            aria-current={view === item.id ? "page" : undefined}
            onclick={() => (view = item.id)}
          >
            <span>{item.label}</span>
            {#if item.count}<span class="rail__count" data-accent={item.id === "updates"}>{item.count}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
    <ul class="rail__list rail__foot">
      <li>
        <button type="button" class="rail__link" aria-current={view === "settings" ? "page" : undefined} onclick={() => (view = "settings")}>
          Settings
        </button>
      </li>
      <li>
        <button type="button" class="rail__link rail__mode" onclick={toggleMode} aria-pressed={dark}>
          {dark ? "Paper" : "Ink"} mode
        </button>
      </li>
    </ul>
    </div>
  </nav>

  <main class="page">
    {#if view === "settings"}
      <header class="head">
        <h1 class="head__title">Settings</h1>
      </header>
      <section class="setting">
        <h2 class="setting__name">Install folder</h2>
        <p class="setting__hint">New installs go here. Apps already installed stay where they are.</p>
        <div class="setting__row">
          <code class="path">{hub.settings.installRoot}</code>
          <button type="button" class="act" onclick={() => hub.chooseInstallRoot()}>Change</button>
        </div>
      </section>
      <section class="setting">
        <h2 class="setting__name">How apps are installed</h2>
        <p class="setting__hint">
          Each app comes from its GitHub release, is checked against the release’s published SHA-256 sums, and is installed
          for your user only — no administrator rights. This computer gets the <code class="path path--inline">{hub.platform}</code>
          builds.
        </p>
      </section>
      <section class="setting">
        <h2 class="setting__name">Credits</h2>
        <p class="setting__hint">
          Crafting Apps Hub is made by
          <a class="anchor" href="https://meatchopper.xyz/portfolio" onclick={(e) => (e.preventDefault(), Browser.OpenURL(e.currentTarget.href))}>Jessada Duangpanyasawang</a>.
          The Craft apps and their icons are made by the ArtCraft Team and contributors at
          <a class="anchor" href="https://github.com/storytold" onclick={(e) => (e.preventDefault(), Browser.OpenURL(e.currentTarget.href))}>storytold</a>,
          licensed MIT or Apache-2.0.
        </p>
      </section>
    {:else}
      <header class="head">
        <div class="head__text">
          <h1 class="head__title">{heading}</h1>
          <p class="head__summary">{summary}</p>
        </div>
        <div class="head__tools">
          <label class="search">
            <SearchIcon class="size-4" aria-hidden="true" />
            <span class="sr-only">Search apps</span>
            <input type="search" placeholder="Search" bind:value={query} />
          </label>
          {#if view === "updates" && hub.updates.length > 1}
            <button type="button" class="act act--accent" disabled={Object.keys(hub.busy).length > 0} onclick={() => hub.updateAll()}>
              Update all
            </button>
          {/if}
          <button type="button" class="act" disabled={hub.refreshing} onclick={() => hub.refresh()}>
            {hub.refreshing ? "Checking" : "Check now"}
          </button>
        </div>
      </header>

      {#if shown.length}
        {#each groups as [group, apps], i (group)}
          <section class="group" aria-labelledby="group-{i}">
            <h2 class="group__title" id="group-{i}">{group} <span class="group__count">{apps.length}</span></h2>
            <ul class="index">
              {#each apps as app (app.id)}
                <AppCard
                  {app}
                  onUninstall={(a) => {
                    cleanUninstall = !a.installed; // leftovers-only removal is always clean
                    uninstallApp = a;
                  }}
                />
              {/each}
            </ul>
          </section>
        {/each}
      {:else}
        <div class="empty">
          <p class="empty__title">
            {query ? `Nothing matches “${query.trim()}”.` : view === "updates" ? "Everything is current." : "Nothing installed yet."}
          </p>
          {#if query}
            <button type="button" class="act" onclick={() => (query = "")}>Clear search</button>
          {:else if view !== "all"}
            <button type="button" class="act" onclick={() => (view = "all")}>See all apps</button>
          {/if}
        </div>
      {/if}
    {/if}

    <footer class="foot">
      <span>Installs to <code class="path path--inline">{hub.settings.installRoot}</code></span>
      <span>{hub.platform}</span>
    </footer>
  </main>
</div>

<!-- Uninstall confirmation -->
<Dialog role="alertdialog" open={!!uninstallApp} onOpenChange={(e) => !e.open && (uninstallApp = null)}>
  <Portal>
    <Dialog.Backdrop class="sheet__backdrop" />
    <Dialog.Positioner class="sheet__pos">
      <Dialog.Content class="sheet">
        <Dialog.Title class="sheet__title">
          {uninstallApp?.installed ? `Uninstall ${uninstallApp.name}?` : `Remove ${uninstallApp?.name} settings?`}
        </Dialog.Title>
        <Dialog.Description class="sheet__body">
          {#if uninstallApp?.installed}
            <p>Removes the program files in</p>
            <code class="path">{uninstallApp.installed.dir}</code>
          {/if}
          <p>Your documents are not touched. Settings stay for a later reinstall unless you tick the box.</p>
        </Dialog.Description>
        {#if uninstallApp?.dataDirs.length}
          <label class="sheet__check">
            <input class="checkbox" type="checkbox" bind:checked={cleanUninstall} disabled={!uninstallApp.installed} />
            <span class="min-w-0">
              Also delete settings, recovery files and caches
              <ul class="sheet__paths">
                {#each uninstallApp.dataDirs as d (d)}<li><code class="path">{d}</code></li>{/each}
              </ul>
            </span>
          </label>
        {/if}
        <footer class="sheet__actions">
          <Dialog.CloseTrigger class="act act--quiet">Cancel</Dialog.CloseTrigger>
          <button
            type="button"
            class="act act--danger"
            onclick={() => {
              const a = uninstallApp;
              uninstallApp = null;
              if (a) hub.uninstall(a, cleanUninstall || !a.installed);
            }}>{uninstallApp?.installed ? "Uninstall" : "Delete"}</button
          >
        </footer>
      </Dialog.Content>
    </Dialog.Positioner>
  </Portal>
</Dialog>

<style>
  .shell {
    display: grid;
    grid-template-columns: 13rem minmax(0, 1fr);
    min-height: 100dvh;
  }

  /* N9 · edge-aligned rail: wordmark + plain text filters, no icons, no fills. */
  .rail {
    position: sticky;
    top: 0;
    height: 100dvh;
    display: flex;
    flex-direction: column;
    gap: 2.5rem;
    padding: 2rem 1.25rem 1.5rem 1.75rem;
    border-right: 1px solid var(--color-rule);
  }
  .wordmark {
    font-family: var(--font-display);
    font-size: 1.5rem;
    font-weight: 700;
    font-variation-settings: "opsz" 72, "SOFT" 50;
    letter-spacing: -0.02em;
    color: var(--color-ink);
  }
  .wordmark span {
    font-weight: 300;
  }
  .rail__list {
    display: grid;
    gap: 0.125rem;
  }
  .rail__foot {
    margin-top: auto;
  }
  .rail__strip {
    display: flex;
    flex: 1;
    flex-direction: column;
  }
  .rail__link {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 0.5rem;
    width: 100%;
    padding: 0.4375rem 0.5rem;
    margin-inline: -0.5rem;
    border-radius: 0.25rem;
    color: var(--color-muted);
    text-align: left;
    white-space: nowrap;
    transition: color var(--dur-short) var(--ease-out);
  }
  .rail__link[aria-current="page"] {
    color: var(--color-ink);
    font-weight: 600;
    text-decoration: underline;
    text-decoration-color: var(--color-accent);
    text-decoration-thickness: 2px;
    text-underline-offset: 0.35em;
  }
  @media (hover: hover) {
    .rail__link:hover {
      color: var(--color-ink);
    }
  }
  .rail__count {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
    color: var(--color-muted);
    text-decoration: none;
  }
  .rail__count[data-accent="true"] {
    color: var(--color-accent);
  }

  .page {
    display: flex;
    flex-direction: column;
    min-width: 0;
    padding: 2.5rem clamp(1.25rem, 4vw, 4rem) 1.5rem;
  }

  /* Index-First header: title + count line on the left, tools hard right. */
  .head {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    justify-content: space-between;
    gap: 1.5rem;
    padding-bottom: 1.5rem;
  }
  .head__text {
    min-width: 0;
  }
  .head__title {
    font-family: var(--font-display);
    font-size: clamp(2rem, 2.5vw + 1rem, 3rem);
    font-weight: 600;
    font-variation-settings: "opsz" 144;
    letter-spacing: -0.025em;
    line-height: 1.05;
    color: var(--color-ink);
    overflow-wrap: anywhere;
  }
  .head__summary {
    margin-top: 0.5rem;
    color: var(--color-muted);
    font-size: 0.9375rem;
  }
  .head__tools {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .search {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    height: 2.5rem;
    padding-inline: 0.875rem;
    border: 1px solid var(--color-rule-2);
    border-radius: 999px;
    color: var(--color-muted);
    transition: background-color var(--dur-short) var(--ease-out);
  }
  .search:focus-within {
    outline: 2px solid var(--color-focus);
    outline-offset: 1px;
  }
  @media (hover: hover) {
    .search:hover {
      background: var(--color-paper-2);
    }
  }
  .search input {
    width: 9rem;
    background: transparent;
    border: 0;
    outline: none;
    padding: 0;
    color: var(--color-ink);
    font-size: 0.875rem;
  }
  .search input::placeholder {
    color: var(--color-muted);
  }

  .index {
    border-bottom: 1px solid var(--color-rule);
  }
  .group + .group {
    margin-top: 2.5rem;
  }
  .group__title {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    padding-bottom: 0.625rem;
    font-family: var(--font-display);
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--color-ink);
  }
  .group__count {
    font-family: var(--font-mono);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
    color: var(--color-muted);
  }

  .empty {
    display: grid;
    justify-items: start;
    gap: 1rem;
    padding-block: 4rem;
    border-top: 1px solid var(--color-rule);
  }
  .empty__title {
    font-family: var(--font-display);
    font-size: 1.5rem;
    font-weight: 400;
    color: var(--color-ink);
  }

  .setting {
    display: grid;
    gap: 0.5rem;
    max-width: 44rem;
    padding-block: 1.5rem;
    border-top: 1px solid var(--color-rule);
  }
  .setting__name {
    font-family: var(--font-display);
    font-size: 1.375rem;
    font-weight: 600;
    color: var(--color-ink);
  }
  .setting__hint {
    max-width: 62ch;
    color: var(--color-muted);
  }
  .setting__row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.5rem;
  }

  .path {
    display: block;
    padding: 0.375rem 0.625rem;
    border-radius: 0.25rem;
    background: var(--color-paper-2);
    color: var(--color-ink);
    font-family: var(--font-mono);
    font-size: 0.8125rem;
    overflow-wrap: anywhere;
    white-space: normal;
  }
  .path--inline {
    display: inline;
    padding: 0.0625rem 0.3125rem;
  }

  /* Ft2 · one quiet line closing the page. */
  .foot {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 0.5rem 1.5rem;
    margin-top: auto;
    padding-top: 3rem;
    color: var(--color-muted);
    font-size: 0.8125rem;
  }

  /* Dialog: a paper sheet, not a floating card. */
  :global(.sheet__backdrop) {
    position: fixed;
    inset: 0;
    z-index: 400;
    background: var(--color-backdrop);
  }
  :global(.sheet__pos) {
    position: fixed;
    inset: 0;
    z-index: 400;
    display: grid;
    place-items: center;
    padding: 1.5rem;
  }
  :global(.sheet) {
    display: grid;
    gap: 1.25rem;
    width: min(100%, 30rem);
    padding: 1.75rem;
    border: 1px solid var(--color-rule-2);
    border-radius: 0.5rem;
    background: var(--color-paper);
    color: var(--color-ink);
    box-shadow: 0 1px 2px var(--color-shadow);
  }
  :global(.sheet__title) {
    font-family: var(--font-display);
    font-size: 1.625rem;
    font-weight: 600;
    letter-spacing: -0.015em;
    line-height: 1.15;
  }
  :global(.sheet__body) {
    display: grid;
    gap: 0.5rem;
    color: var(--color-muted);
    font-size: 0.9375rem;
  }
  .sheet__check {
    display: flex;
    align-items: flex-start;
    gap: 0.625rem;
    font-size: 0.9375rem;
  }
  .sheet__check input {
    margin-top: 0.1875rem;
    accent-color: var(--color-accent);
  }
  .sheet__paths {
    display: grid;
    gap: 0.25rem;
    margin-top: 0.5rem;
  }
  .sheet__actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }
  :global(.act--danger) {
    border-color: var(--color-danger-fill);
    background: var(--color-danger-fill);
    color: var(--color-danger-ink);
  }
  @media (hover: hover) {
    :global(.act--danger:hover) {
      background: var(--color-ink);
      border-color: var(--color-ink);
    }
  }

  /* Narrow: rail becomes a top strip; filters stay one line. */
  @media (max-width: 48rem) {
    .shell {
      grid-template-columns: minmax(0, 1fr);
    }
    .rail {
      position: static;
      height: auto;
      flex-direction: column;
      align-items: stretch;
      gap: 0.75rem;
      padding: 1rem 1.25rem 0;
      border-right: 0;
      border-bottom: 1px solid var(--color-rule);
    }
    /* One scrollable strip of filters + Settings: never wraps to a lonely row. */
    .rail__list,
    .rail__foot {
      display: contents;
    }
    .rail__strip {
      flex-direction: row;
      align-items: baseline;
      gap: 1.25rem;
      overflow-x: auto;
      scrollbar-width: none;
      padding-bottom: 0.75rem;
    }
    .rail__strip li {
      flex: none;
    }
    .rail__link {
      width: auto;
      margin-inline: 0;
      padding-inline: 0;
    }
    .rail__mode {
      display: none;
    }
    .head__tools {
      width: 100%;
    }
    .search {
      flex: 1;
    }
    .search input {
      width: 100%;
    }
  }
</style>
