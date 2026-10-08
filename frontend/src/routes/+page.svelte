<script lang="ts">
  import { onMount } from "svelte";
  import { Browser } from "@wailsio/runtime";
  import { Dialog, Navigation, Portal } from "@skeletonlabs/skeleton-svelte";
  import LayoutGridIcon from "@lucide/svelte/icons/layout-grid";
  import HardDriveIcon from "@lucide/svelte/icons/hard-drive";
  import ArrowUpCircleIcon from "@lucide/svelte/icons/circle-arrow-up";
  import SettingsIcon from "@lucide/svelte/icons/settings";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import SearchIcon from "@lucide/svelte/icons/search";
  import SunIcon from "@lucide/svelte/icons/sun";
  import MoonIcon from "@lucide/svelte/icons/moon";
  import XIcon from "@lucide/svelte/icons/x";
  import ExternalLinkIcon from "@lucide/svelte/icons/external-link";
  import FolderCogIcon from "@lucide/svelte/icons/folder-cog";
  import PackageOpenIcon from "@lucide/svelte/icons/package-open";
  import type { AppState } from "#bindings/craftinghub/internal/hub/models.ts";
  import AppCard from "#lib/AppCard.svelte";
  import { hub } from "#lib/hub.svelte.ts";

  type View = "all" | "installed" | "updates" | "settings";
  let view = $state<View>("all");
  let query = $state("");
  let dark = $state(true);
  let notesApp = $state<AppState | null>(null);
  let uninstallApp = $state<AppState | null>(null);
  let cleanUninstall = $state(false);

  const shown = $derived(
    (view === "installed" ? hub.installed : view === "updates" ? hub.updates : hub.apps).filter((a) =>
      `${a.name} ${a.category} ${a.description}`.toLowerCase().includes(query.trim().toLowerCase()),
    ),
  );

  const nav = $derived([
    { id: "all" as const, label: "Library", icon: LayoutGridIcon, count: hub.apps.length },
    { id: "installed" as const, label: "Installed", icon: HardDriveIcon, count: hub.installed.length },
    { id: "updates" as const, label: "Updates", icon: ArrowUpCircleIcon, count: hub.updates.length },
  ]);

  const animation =
    "transition transition-discrete opacity-0 translate-y-4 starting:data-[state=open]:opacity-0 starting:data-[state=open]:translate-y-4 data-[state=open]:opacity-100 data-[state=open]:translate-y-0";

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

<div class="h-full grid grid-cols-[auto_1fr]">
  <Navigation layout="rail" class="border-r border-surface-200-800">
    <Navigation.Header>
      <img src="/icons/photocraft.png" alt="Crafting Apps Hub" class="size-10 rounded-lg mx-auto my-2" />
    </Navigation.Header>
    <Navigation.Content>
      <Navigation.Menu>
        {#each nav as item (item.id)}
          {@const Icon = item.icon}
          <Navigation.Trigger
            onclick={() => (view = item.id)}
            aria-current={view === item.id ? "page" : undefined}
            class="relative {view === item.id ? 'preset-filled-primary-500' : ''}"
          >
            <Icon class="size-5" />
            <Navigation.TriggerText>{item.label}</Navigation.TriggerText>
            {#if item.id === "updates" && item.count}
              <span class="badge-icon preset-filled-warning-500 absolute top-1 right-1 size-5 text-xs">{item.count}</span>
            {/if}
          </Navigation.Trigger>
        {/each}
      </Navigation.Menu>
    </Navigation.Content>
    <Navigation.Footer>
      <Navigation.Trigger onclick={toggleMode} title="Toggle light/dark" aria-label="Toggle light or dark mode">
        {#if dark}<SunIcon class="size-5" />{:else}<MoonIcon class="size-5" />{/if}
      </Navigation.Trigger>
      <Navigation.Trigger
        onclick={() => (view = "settings")}
        title="Settings"
        aria-label="Settings"
        aria-current={view === "settings" ? "page" : undefined}
        class={view === "settings" ? "preset-filled-primary-500" : ""}
      >
        <SettingsIcon class="size-5" />
      </Navigation.Trigger>
    </Navigation.Footer>
  </Navigation>

  <main class="overflow-y-auto">
    {#if view === "settings"}
      <section class="max-w-2xl p-8 space-y-6">
        <h1 class="h3">Settings</h1>
        <div class="card preset-filled-surface-100-900 border border-surface-200-800 p-5 space-y-3">
          <h2 class="h6 flex items-center gap-2"><FolderCogIcon class="size-5" />Install folder</h2>
          <p class="text-sm opacity-70">
            New installs go here. Apps already installed stay where they are; reinstall to move them.
          </p>
          <div class="flex gap-2">
            <input class="input flex-1 font-mono text-sm" readonly value={hub.settings.installRoot} aria-label="Install folder" />
            <button type="button" class="btn preset-tonal" onclick={() => hub.chooseInstallRoot()}>Change…</button>
          </div>
        </div>
        <div class="card preset-filled-surface-100-900 border border-surface-200-800 p-5 space-y-2 text-sm">
          <h2 class="h6">About</h2>
          <p class="opacity-70">
            Platform <code class="code">{hub.platform}</code>. Apps are downloaded from their GitHub releases, verified
            against the published <code class="code">SHA256SUMS.txt</code>, and installed per-user (no admin rights).
          </p>
        </div>
      </section>
    {:else}
      <header class="sticky top-0 z-10 bg-surface-50-950/90 backdrop-blur border-b border-surface-200-800 px-8 py-4 flex flex-wrap items-center gap-4">
        <div class="flex-1 min-w-48">
          <h1 class="h4">{nav.find((n) => n.id === view)?.label}</h1>
          <p class="text-sm opacity-60">
            {hub.installed.length} of {hub.apps.length} installed{hub.updates.length ? ` · ${hub.updates.length} update${hub.updates.length > 1 ? "s" : ""} available` : ""}
          </p>
        </div>
        <div class="field-group grid-cols-[auto_1fr] w-64">
          <label class="label label-text preset-tonal" for="search" aria-label="Search"><SearchIcon class="size-4" /></label>
          <input class="input" id="search" type="search" placeholder="Search apps" bind:value={query} />
        </div>
        {#if hub.updates.length > 1}
          <button type="button" class="btn preset-filled-warning-500" disabled={Object.keys(hub.busy).length > 0} onclick={() => hub.updateAll()}>
            <ArrowUpCircleIcon class="size-4" />Update all
          </button>
        {/if}
        <button type="button" class="btn preset-tonal" disabled={hub.refreshing} onclick={() => hub.refresh()}>
          <RefreshCwIcon class="size-4 {hub.refreshing ? 'animate-spin' : ''}" />Check for updates
        </button>
      </header>

      {#if shown.length}
        <div class="grid gap-5 p-8 grid-cols-[repeat(auto-fill,minmax(19rem,1fr))]">
          {#each shown as app (app.id)}
            <AppCard
              {app}
              onNotes={(a) => (notesApp = a)}
              onUninstall={(a) => {
                cleanUninstall = !a.installed; // leftovers-only removal is always clean
                uninstallApp = a;
              }}
            />
          {/each}
        </div>
      {:else}
        <div class="flex flex-col items-center justify-center gap-3 py-24 opacity-70">
          <PackageOpenIcon class="size-12" />
          <p>
            {query ? "No apps match your search." : view === "updates" ? "Everything is up to date." : "No apps installed yet."}
          </p>
          {#if view !== "all" && !query}
            <button type="button" class="btn preset-tonal" onclick={() => (view = "all")}>Browse library</button>
          {/if}
        </div>
      {/if}
    {/if}
  </main>
</div>

<!-- Release notes -->
<Dialog open={!!notesApp} onOpenChange={(e) => !e.open && (notesApp = null)}>
  <Portal>
    <Dialog.Backdrop class="fixed inset-0 z-50 bg-surface-50-950/60" />
    <Dialog.Positioner class="fixed inset-0 z-50 flex justify-center items-center p-6">
      <Dialog.Content class="card bg-surface-100-900 w-full max-w-2xl max-h-[85vh] flex flex-col p-5 gap-4 shadow-xl {animation}">
        <header class="flex justify-between items-center gap-4">
          <Dialog.Title class="h5">{notesApp?.name} v{notesApp?.latestVersion}</Dialog.Title>
          <Dialog.CloseTrigger class="btn-icon hover:preset-tonal" aria-label="Close"><XIcon class="size-4" /></Dialog.CloseTrigger>
        </header>
        <pre class="pre text-xs whitespace-pre-wrap overflow-y-auto flex-1">{notesApp?.releaseNotes}</pre>
        <footer class="flex justify-end">
          <button type="button" class="btn preset-tonal" onclick={() => notesApp && Browser.OpenURL(notesApp.releaseUrl)}>
            <ExternalLinkIcon class="size-4" />View on GitHub
          </button>
        </footer>
      </Dialog.Content>
    </Dialog.Positioner>
  </Portal>
</Dialog>

<!-- Uninstall confirmation -->
<Dialog role="alertdialog" open={!!uninstallApp} onOpenChange={(e) => !e.open && (uninstallApp = null)}>
  <Portal>
    <Dialog.Backdrop class="fixed inset-0 z-50 bg-surface-50-950/60" />
    <Dialog.Positioner class="fixed inset-0 z-50 flex justify-center items-center p-6">
      <Dialog.Content class="card bg-surface-100-900 w-full max-w-md p-5 space-y-4 shadow-xl {animation}">
        <Dialog.Title class="h5">
          {uninstallApp?.installed ? `Uninstall ${uninstallApp.name}?` : `Remove ${uninstallApp?.name} settings?`}
        </Dialog.Title>
        <Dialog.Description class="text-sm opacity-80 space-y-2">
          {#if uninstallApp?.installed}
            <p>Removes the program files in</p>
            <code class="code block break-all whitespace-normal">{uninstallApp.installed.dir}</code>
          {/if}
          <p>Your documents are not affected. Settings are kept for a later reinstall unless you tick the box below.</p>
        </Dialog.Description>
        {#if uninstallApp?.dataDirs.length}
          <label class="flex items-start gap-2 text-sm">
            <input class="checkbox mt-0.5" type="checkbox" bind:checked={cleanUninstall} disabled={!uninstallApp.installed} />
            <span class="min-w-0">
              Also delete settings, recovery files and caches
              <ul class="mt-1 space-y-0.5 text-xs opacity-70">
                {#each uninstallApp.dataDirs as d (d)}<li><code class="code block break-all whitespace-normal">{d}</code></li>{/each}
              </ul>
            </span>
          </label>
        {/if}
        <footer class="flex justify-end gap-2">
          <Dialog.CloseTrigger class="btn preset-tonal">Cancel</Dialog.CloseTrigger>
          <button
            type="button"
            class="btn preset-filled-error-500"
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
