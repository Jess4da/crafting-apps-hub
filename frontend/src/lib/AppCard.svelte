<script lang="ts">
  /* Hallmark · catalogue row (Index-First) · tokens from routes/atelier.css + layout.css */
  import { Menu, Portal, Progress } from "@skeletonlabs/skeleton-svelte";
  import { Browser } from "@wailsio/runtime";
  import EllipsisIcon from "@lucide/svelte/icons/ellipsis";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import type { AppState } from "#bindings/craftinghub/internal/hub/models.ts";
  import { hub, formatBytes } from "./hub.svelte";

  let { app, onUninstall }: { app: AppState; onUninstall: (a: AppState) => void } = $props();

  const busy = $derived(hub.busy[app.id]);
  const progress = $derived(hub.progress[app.id]);
  const percent = $derived(progress?.total ? Math.round((progress.received / progress.total) * 100) : null);
  const stageText = $derived(
    !progress
      ? "Preparing"
      : progress.stage === "download"
        ? `${formatBytes(progress.received)} of ${formatBytes(progress.total)}`
        : progress.stage === "verify"
          ? "Checking signature"
          : "Unpacking",
  );
  // One sentence of status under the name: the row's single source of truth.
  const status = $derived(
    app.error && !app.latestVersion
      ? { tone: "error", text: app.error }
      : app.latestVersion && !app.supported
        ? { tone: "warn", text: `No build for ${hub.platform} yet` }
        : null,
  );
  const hasMenu = $derived(!!app.installed || app.dataDirs.length > 0 || !!app.releaseUrl);
</script>

<li class="row" aria-busy={!!busy}>
  <img src="/icons/{app.id}.png" alt="" width="56" height="56" class="row__icon" />

  <div class="row__text">
    <h3 class="row__name">{app.name}</h3>
    <p class="row__desc">{app.description}</p>
    {#if busy === "install"}
      <Progress value={percent} class="row__progress">
        <Progress.Label class="row__meta">{stageText}</Progress.Label>
        <Progress.Track class="row__track">
          <Progress.Range class="row__range" />
        </Progress.Track>
      </Progress>
    {:else if status}
      <p class="row__meta" data-tone={status.tone}>{status.text}</p>
    {/if}
  </div>

  <p class="row__version">
    {#if app.installed}
      <span>v{app.installed.version}</span>
      {#if app.updateAvailable}<span class="row__next">→ v{app.latestVersion}</span>{/if}
    {:else if app.latestVersion}
      <span>v{app.latestVersion}</span>
      {#if app.downloadSize}<span class="row__size">{formatBytes(app.downloadSize)}</span>{/if}
    {:else if hub.refreshing}
      <span class="row__size">checking</span>
    {/if}
  </p>

  <div class="row__actions">
    {#if busy}
      <span class="act act--quiet" role="status">
        <LoaderCircleIcon class="size-4 animate-spin" />{busy === "install" ? "Working" : "Removing"}
      </span>
    {:else if app.updateAvailable}
      <button type="button" class="act act--accent" onclick={() => hub.install(app)}>Update</button>
    {:else if app.installed}
      <button type="button" class="act" onclick={() => hub.launch(app)}>Open</button>
    {:else}
      <button type="button" class="act" disabled={!app.supported} onclick={() => hub.install(app)}>Install</button>
    {/if}

    {#if hasMenu}
      <Menu
        onSelect={(e) => {
          if (e.value === "open") hub.launch(app);
          else if (e.value === "folder") hub.openFolder(app.id);
          else if (e.value === "notes") Browser.OpenURL(app.releaseUrl);
          else if (e.value === "remove") onUninstall(app);
        }}
      >
        <Menu.Trigger class="more" aria-label="More for {app.name}" disabled={!!busy}>
          <EllipsisIcon class="size-4" />
        </Menu.Trigger>
        <Portal>
          <Menu.Positioner>
            <Menu.Content class="menu">
              {#if app.installed && app.updateAvailable}
                <Menu.Item value="open"><Menu.ItemText>Open v{app.installed.version}</Menu.ItemText></Menu.Item>
              {/if}
              {#if app.installed}
                <Menu.Item value="folder"><Menu.ItemText>Show in folder</Menu.ItemText></Menu.Item>
              {/if}
              {#if app.releaseUrl}
                <Menu.Item value="notes"><Menu.ItemText>Release notes</Menu.ItemText></Menu.Item>
              {/if}
              {#if app.installed || app.dataDirs.length}
                <Menu.Separator />
                <Menu.Item value="remove" class="menu__danger">
                  <Menu.ItemText>{app.installed ? "Uninstall…" : "Remove leftover settings…"}</Menu.ItemText>
                </Menu.Item>
              {/if}
            </Menu.Content>
          </Menu.Positioner>
        </Portal>
      </Menu>
    {/if}
  </div>
</li>

<style>
  .row {
    display: grid;
    grid-template-columns: 3.5rem minmax(0, 1fr) auto auto;
    align-items: center;
    column-gap: 1.25rem;
    padding-block: 1.125rem;
    border-top: 1px solid var(--color-rule);
  }
  .row__icon {
    width: 3.5rem;
    height: 3.5rem;
    border-radius: 0.625rem;
  }
  .row__text {
    min-width: 0;
  }
  .row__name {
    font-family: var(--font-display);
    font-size: 1.375rem;
    font-weight: 600;
    font-variation-settings: "opsz" 48;
    letter-spacing: -0.01em;
    line-height: 1.15;
    color: var(--color-ink);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .row__desc {
    margin-top: 0.125rem;
    max-width: 62ch;
    color: var(--color-muted);
    font-size: 0.9375rem;
  }
  .row__meta {
    margin-top: 0.375rem;
    font-size: 0.8125rem;
    color: var(--color-muted);
  }
  .row__meta[data-tone="error"] {
    color: var(--color-danger);
  }
  .row__meta[data-tone="warn"] {
    color: var(--color-warn-text);
  }
  :global(.row__progress) {
    margin-top: 0.5rem;
    display: grid;
    gap: 0.25rem;
    max-width: 24rem;
  }
  :global(.row__track) {
    height: 0.25rem;
    background: var(--color-rule);
  }
  :global(.row__range) {
    background: var(--color-progress);
  }
  .row__version {
    display: grid;
    justify-items: end;
    font-family: var(--font-mono);
    font-variant-numeric: tabular-nums;
    font-size: 0.8125rem;
    color: var(--color-ink);
    white-space: nowrap;
  }
  .row__next {
    color: var(--color-accent);
  }
  .row__size {
    color: var(--color-muted);
  }
  .row__actions {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  /* Narrow windows: drop version + description; name truncates before the button. */
  @media (max-width: 40rem) {
    .row {
      grid-template-columns: 2.75rem minmax(0, 1fr) auto;
      column-gap: 0.75rem;
    }
    .row__icon {
      width: 2.75rem;
      height: 2.75rem;
    }
    .row__version,
    .row__desc {
      display: none;
    }
    .row__name {
      font-size: 1.125rem;
    }
    .row__actions :global(.act) {
      min-width: 0;
      padding-inline: 0.875rem;
    }
  }
</style>
