<script lang="ts">
  /* Hallmark · genre: liquid glass · chrome floats on the OS material (rail, strip, search, footer).
   * Styles here reference tokens from routes/layout.css only. */
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
  let uninstallApp = $state<AppState | null>(null);
  let cleanUninstall = $state(false);
  let defaultsApp = $state<AppState | null>(null);
  let defaultExts = $state<string[]>([]);

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
    hub.init();
  });
</script>

<div class="shell">
  <nav class="rail glass" aria-label="Library">
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
          <a class="anchor" href="https://github.com/Jess4da" onclick={(e) => (e.preventDefault(), Browser.OpenURL(e.currentTarget.href))}>Jess4da</a>
          (<a class="anchor" href="https://github.com/Jess4da/crafting-apps-hub" onclick={(e) => (e.preventDefault(), Browser.OpenURL(e.currentTarget.href))}>source</a>, MIT).
          The Craft apps and their icons are made by the ArtCraft Team and contributors at
          <a class="anchor" href="https://github.com/storytold" onclick={(e) => (e.preventDefault(), Browser.OpenURL(e.currentTarget.href))}>storytold</a>,
          licensed MIT or Apache-2.0.
        </p>
      </section>
    {:else}
      <header class="head">
        <div class="head__text">
          <h1 class="head__title">{heading}</h1>
          {#if hub.loading}
            <p class="head__summary"><span class="placeholder motion-safe:animate-pulse ghost__bar" style:width="14rem"></span></p>
          {:else}
            <p class="head__summary">{summary}</p>
          {/if}
        </div>
        <div class="head__tools">
          <label class="search glass">
            <SearchIcon class="size-4" aria-hidden="true" />
            <span class="sr-only">Search apps</span>
            <input type="search" placeholder="Search" bind:value={query} />
          </label>
          {#if hub.updates.length}
            <button type="button" class="act act--accent" disabled={Object.keys(hub.busy).length > 0} onclick={() => hub.updateAll()}>
              Update all
            </button>
          {/if}
          <button type="button" class="act" disabled={hub.refreshing} onclick={() => hub.refresh()}>
            {hub.refreshing ? "Checking" : "Check now"}
          </button>
        </div>
      </header>

      {#if hub.loading}
        <p class="sr-only" role="status">Loading apps</p>
        <ul class="index" aria-hidden="true">
          {#each [40, 30, 45, 35, 40, 30] as w, i (i)}
            <li class="ghost">
              <span class="placeholder motion-safe:animate-pulse ghost__icon"></span>
              <span class="ghost__text">
                <span class="placeholder motion-safe:animate-pulse ghost__bar" style:width="{w}%"></span>
                <span class="placeholder motion-safe:animate-pulse ghost__bar" style:width="{w + 30}%"></span>
              </span>
              <span class="placeholder motion-safe:animate-pulse ghost__act"></span>
            </li>
          {/each}
        </ul>
      {:else if shown.length}
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
                  onDefaults={(a) => {
                    // First time: everything ticked; later: the saved choice.
                    defaultExts = a.installed?.defaultExts?.length ? [...a.installed.defaultExts] : [...a.extensions];
                    defaultsApp = a;
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

    <footer class="foot glass" hidden={hub.loading}>
      <span>Installs to <code class="path path--inline">{hub.settings.installRoot}</code></span>
      <span class="foot__right">
        {#if hub.hubUpdate}
          {@const u = hub.hubUpdate}
          <span class="foot__update">
            {#if !u.inPlace}
              <button type="button" class="anchor" onclick={() => Browser.OpenURL(u.url)}>Hub v{u.version} available</button>
            {:else if hub.hubStage === "ready"}
              Hub v{u.version} ready
              <button type="button" class="act act--accent" disabled={Object.keys(hub.busy).length > 0} onclick={() => hub.restartHub()}>Restart</button>
            {:else if hub.hubStage === "downloading"}
              Downloading hub <span class="foot__ver">{hub.hubProgress.total ? Math.round((hub.hubProgress.received / hub.hubProgress.total) * 100) : 0}%</span>
            {:else}
              Hub v{u.version}
              <button type="button" class="act act--accent" onclick={() => hub.installHubUpdate()}>Update</button>
            {/if}
          </span>
        {/if}
        <span><span class="foot__ver">v{hub.version}</span> · {hub.platform}</span>
      </span>
    </footer>
  </main>
</div>

<!-- Uninstall confirmation -->
<Dialog role="alertdialog" open={!!uninstallApp} onOpenChange={(e) => !e.open && (uninstallApp = null)}>
  <Portal>
    <Dialog.Backdrop class="sheet__backdrop" />
    <Dialog.Positioner class="sheet__pos">
      <Dialog.Content class="sheet glass--float">
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

<!-- Default app: pick which file types open in this app -->
<Dialog open={!!defaultsApp} onOpenChange={(e) => !e.open && (defaultsApp = null)}>
  <Portal>
    <Dialog.Backdrop class="sheet__backdrop" />
    <Dialog.Positioner class="sheet__pos">
      <Dialog.Content class="sheet glass--float">
        <Dialog.Title class="sheet__title">Open files with {defaultsApp?.name}</Dialog.Title>
        <Dialog.Description class="sheet__body">
          <p>Ticked types open in {defaultsApp?.name}. Unticked ones go back to the app that opened them before.</p>
        </Dialog.Description>
        {#if defaultsApp}
          {@const all = defaultsApp.extensions}
          <label class="sheet__check">
            <input
              class="checkbox"
              type="checkbox"
              checked={defaultExts.length === all.length}
              indeterminate={defaultExts.length > 0 && defaultExts.length < all.length}
              onchange={(e) => (defaultExts = e.currentTarget.checked ? [...all] : [])}
            />
            <span>All file types</span>
          </label>
          <div class="sheet__exts">
            {#each all as x (x)}
              <label class="sheet__check">
                <input class="checkbox" type="checkbox" value={x} bind:group={defaultExts} />
                <code class="path">.{x}</code>
              </label>
            {/each}
          </div>
        {/if}
        <footer class="sheet__actions">
          <Dialog.CloseTrigger class="act act--quiet">Cancel</Dialog.CloseTrigger>
          <button
            type="button"
            class="act"
            disabled={!defaultExts.length && !defaultsApp?.installed?.defaultExts?.length}
            onclick={() => {
              const a = defaultsApp;
              defaultsApp = null;
              // Keep catalog order so the saved list is stable.
              if (a) hub.setDefault(a, a.extensions.filter((x) => defaultExts.includes(x)));
            }}>Save</button
          >
        </footer>
      </Dialog.Content>
    </Dialog.Positioner>
  </Portal>
</Dialog>

<style>
  /* The window is the backdrop: chrome floats 8px inside its edges. */
  .shell {
    display: grid;
    grid-template-columns: 13rem minmax(0, 1fr);
    gap: 0.5rem;
    min-height: 100dvh;
    padding: 0.5rem;
  }

  /* Rail: a floating glass panel, inset from the window edges. */
  .rail {
    position: sticky;
    top: 0.5rem;
    height: calc(100dvh - 1rem);
    display: flex;
    flex-direction: column;
    gap: 2rem;
    padding: 1.5rem 1rem;
    border-radius: 1.375rem;
    overflow-y: auto;
  }
  .wordmark {
    padding-inline: 0.75rem;
    font-family: var(--font-body);
    font-size: 1.25rem;
    font-weight: 600;
    letter-spacing: -0.02em;
    color: var(--color-ink);
  }
  .wordmark span {
    font-weight: 400;
    color: var(--color-muted);
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
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    border-radius: 999px;
    color: var(--color-muted);
    text-align: left;
    white-space: nowrap;
    transition:
      background-color var(--dur-short) var(--ease-out),
      color var(--dur-short) var(--ease-out);
  }
  .rail__link[aria-current="page"] {
    background-image: linear-gradient(var(--glass-hover), var(--glass-hover));
    color: var(--color-ink);
    font-weight: 600;
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
    gap: 2rem;
    min-width: 0;
    min-height: calc(100dvh - 1rem);
    padding: 2rem clamp(1.25rem, 4vw, 3.5rem) 1.5rem;
    border-radius: 1.375rem;
  }

  /* Header: title + count line left, glass tools right. */
  .head {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    justify-content: space-between;
    gap: 1.5rem;
  }
  .head__text {
    min-width: 0;
  }
  .head__title {
    font-family: var(--font-body);
    font-size: clamp(1.75rem, 2vw + 1rem, 2.5rem);
    font-weight: 600;
    letter-spacing: -0.02em;
    line-height: 1.1;
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
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
  }
  .search {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
    height: 2.5rem;
    padding-inline: 0.875rem;
    border-radius: 999px;
    color: var(--color-muted);
    transition: background-color var(--dur-short) var(--ease-out);
  }
  .search:focus-within {
    outline: 2px solid var(--color-focus);
    outline-offset: 2px;
  }
  @media (hover: hover) {
    .search:hover {
      background-image: linear-gradient(var(--glass-hover), var(--glass-hover));
    }
  }
  .search input {
    min-width: 0;
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
  /* Loading skeleton: same grid and sizes as AppCard's .row, so nothing shifts on load. */
  .ghost {
    display: grid;
    grid-template-columns: 3.5rem minmax(0, 1fr) auto;
    align-items: center;
    column-gap: 1.25rem;
    padding-block: 1.125rem;
    border-top: 1px solid var(--color-rule);
  }
  .ghost__icon {
    width: 3.5rem;
    height: 3.5rem;
    border-radius: 0.875rem;
  }
  .ghost__text {
    display: grid;
    gap: 0.5rem;
  }
  .ghost__bar {
    display: block;
    max-width: 100%;
    min-height: 0.875rem;
  }
  .ghost__act {
    width: 6rem;
    height: 2.5rem;
    border-radius: 999px;
  }
  @media (max-width: 40rem) {
    .ghost {
      grid-template-columns: 2.75rem minmax(0, 1fr) auto;
      column-gap: 0.75rem;
    }
    .ghost__icon {
      width: 2.75rem;
      height: 2.75rem;
    }
    .ghost__act {
      width: 4.5rem;
    }
  }
  .group__title {
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    padding-bottom: 0.625rem;
    font-family: var(--font-body);
    font-size: 1rem;
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
    font-family: var(--font-body);
    font-size: 1.5rem;
    font-weight: 600;
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
    font-family: var(--font-body);
    font-size: 1.25rem;
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
    border-radius: 0.375rem;
    background: var(--glass-fill);
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

  /* Footer: one quiet glass bar closing the page. */
  .foot {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem 1.5rem;
    margin-top: auto;
    padding: 0.625rem 1rem;
    border-radius: 1.125rem;
    color: var(--color-muted);
    font-size: 0.8125rem;
  }
  .foot[hidden] {
    display: none;
  }
  .foot__ver {
    font-family: var(--font-mono);
  }
  .foot__right {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.5rem 1rem;
  }
  .foot__update {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--color-accent);
  }

  /* Dialog: a floating glass sheet over a scrim. The sheet blurs; the scrim never does. */
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
    border-radius: 1.75rem;
    color: var(--color-ink);
  }
  :global(.sheet__title) {
    font-family: var(--font-body);
    font-size: 1.375rem;
    font-weight: 600;
    letter-spacing: -0.015em;
    line-height: 1.2;
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
    accent-color: var(--color-accent-fill);
  }
  .sheet__paths {
    display: grid;
    gap: 0.25rem;
    margin-top: 0.5rem;
  }
  .sheet__exts {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(6.5rem, 1fr));
    gap: 0.5rem 1rem;
    max-height: 16rem;
    overflow-y: auto;
    padding-left: 1.625rem;
  }
  .sheet__actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  /* Narrow: the rail becomes one horizontal glass pill bar; filters scroll, never wrap. */
  @media (max-width: 48rem) {
    .shell {
      grid-template-columns: minmax(0, 1fr);
    }
    .rail {
      height: auto;
      flex-direction: row;
      align-items: center;
      gap: 0;
      padding: 0.375rem 0.5rem;
      overflow: hidden;
      /* Sticky over scrolled rows: a strong fill plus blur keeps the muted
       * labels legible. The rail is not inside another blurred surface. */
      background: var(--glass-fill-strong);
      -webkit-backdrop-filter: blur(var(--glass-blur));
      backdrop-filter: blur(var(--glass-blur));
      z-index: 1;
    }
    /* macOS draws the traffic lights over the hidden title bar (50 px); keep
     * the rail's content clear of them. */
    :global(html.mac) .rail {
      padding-top: 2.75rem;
    }
    .wordmark {
      display: none;
    }
    .rail__strip {
      flex: 1;
      flex-direction: row;
      align-items: center;
      overflow-x: auto;
      scrollbar-width: none;
      /* Room for :focus-visible outlines, which overflow would clip. */
      padding: 4px;
    }
    .rail__list,
    .rail__foot {
      display: contents;
    }
    .rail__strip li {
      flex: none;
    }
    .rail__link {
      width: auto;
      padding-inline: 0.75rem;
    }
    .head__tools {
      width: 100%;
    }
    .search {
      flex: 1 1 10rem;
    }
    .search input {
      width: 100%;
    }
  }
</style>
