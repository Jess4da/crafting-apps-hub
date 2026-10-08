<script lang="ts">
  import { Progress } from "@skeletonlabs/skeleton-svelte";
  import DownloadIcon from "@lucide/svelte/icons/download";
  import RefreshCwIcon from "@lucide/svelte/icons/refresh-cw";
  import PlayIcon from "@lucide/svelte/icons/play";
  import FolderOpenIcon from "@lucide/svelte/icons/folder-open";
  import Trash2Icon from "@lucide/svelte/icons/trash-2";
  import EraserIcon from "@lucide/svelte/icons/eraser";
  import ScrollTextIcon from "@lucide/svelte/icons/scroll-text";
  import CircleAlertIcon from "@lucide/svelte/icons/circle-alert";
  import LoaderCircleIcon from "@lucide/svelte/icons/loader-circle";
  import type { AppState } from "#bindings/craftinghub/internal/hub/models.ts";
  import { hub, formatBytes } from "./hub.svelte";

  let {
    app,
    onNotes,
    onUninstall,
  }: { app: AppState; onNotes: (a: AppState) => void; onUninstall: (a: AppState) => void } = $props();

  const busy = $derived(hub.busy[app.id]);
  const progress = $derived(hub.progress[app.id]);
  const percent = $derived(progress?.total ? Math.round((progress.received / progress.total) * 100) : null);
  const stageText = $derived(
    !progress
      ? "Preparing…"
      : progress.stage === "download"
        ? `Downloading ${formatBytes(progress.received)} of ${formatBytes(progress.total)}`
        : progress.stage === "verify"
          ? "Verifying checksum…"
          : progress.stage === "extract"
            ? "Installing…"
            : "Finishing…",
  );
</script>

<article class="card preset-filled-surface-100-900 border border-surface-200-800 flex flex-col gap-4 p-5">
  <header class="flex items-start gap-4">
    <img src="/icons/{app.id}.png" alt="" class="size-14 rounded-xl shadow" />
    <div class="min-w-0 flex-1">
      <h3 class="h5 truncate">{app.name}</h3>
      <p class="text-sm opacity-70">{app.category}</p>
    </div>
    {#if app.updateAvailable}
      <span class="badge preset-filled-warning-500 shrink-0">Update</span>
    {:else if app.installed}
      <span class="badge preset-tonal-success shrink-0">Installed</span>
    {/if}
  </header>

  <p class="text-sm opacity-80 flex-1">{app.description}</p>

  <dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
    {#if app.installed}
      <dt class="opacity-60">Installed</dt>
      <dd>v{app.installed.version}</dd>
    {/if}
    <dt class="opacity-60">Latest</dt>
    <dd>
      {#if app.latestVersion}
        v{app.latestVersion}
        {#if app.downloadSize}<span class="opacity-60">· {formatBytes(app.downloadSize)}</span>{/if}
      {:else if hub.refreshing}
        <span class="opacity-60">checking…</span>
      {:else}
        <span class="opacity-60">unknown</span>
      {/if}
    </dd>
  </dl>

  {#if app.error && !app.latestVersion}
    <p class="text-xs text-error-600-400 flex gap-1 items-start">
      <CircleAlertIcon class="size-4 shrink-0" />{app.error}
    </p>
  {:else if app.latestVersion && !app.supported}
    <p class="text-xs text-warning-600-400 flex gap-1 items-start">
      <CircleAlertIcon class="size-4 shrink-0" />No build for {hub.platform}
    </p>
  {/if}

  {#if busy === "install"}
    <Progress value={percent} class="space-y-1">
      <Progress.Label class="text-xs opacity-80">{stageText}</Progress.Label>
      <Progress.Track class="h-2">
        <Progress.Range class="bg-primary-500" />
      </Progress.Track>
    </Progress>
  {/if}

  <footer class="flex items-center gap-2">
    {#if app.installed}
      <button type="button" class="btn preset-filled-primary-500 flex-1" disabled={!!busy} onclick={() => hub.launch(app)}>
        <PlayIcon class="size-4" />Open
      </button>
      {#if app.updateAvailable}
        <button type="button" class="btn preset-filled-warning-500" disabled={!!busy} onclick={() => hub.install(app)}>
          <RefreshCwIcon class="size-4 {busy === 'install' ? 'animate-spin' : ''}" />Update
        </button>
      {/if}
      <button type="button" class="btn-icon preset-tonal" title="Show in folder" aria-label="Show {app.name} in folder" onclick={() => hub.openFolder(app.id)}>
        <FolderOpenIcon class="size-4" />
      </button>
      <button type="button" class="btn-icon preset-tonal hover:preset-filled-error-500" title="Uninstall" aria-label="Uninstall {app.name}" disabled={!!busy} onclick={() => onUninstall(app)}>
        {#if busy === "uninstall"}<LoaderCircleIcon class="size-4 animate-spin" />{:else}<Trash2Icon class="size-4" />{/if}
      </button>
    {:else}
      <button type="button" class="btn preset-filled-primary-500 flex-1" disabled={!!busy || !app.supported} onclick={() => hub.install(app)}>
        {#if busy}<LoaderCircleIcon class="size-4 animate-spin" />Installing{:else}<DownloadIcon class="size-4" />Install{/if}
      </button>
      {#if app.dataDirs.length}
        <button type="button" class="btn-icon preset-tonal hover:preset-filled-error-500" title="Remove leftover settings" aria-label="Remove {app.name} leftover settings" disabled={!!busy} onclick={() => onUninstall(app)}>
          {#if busy === "uninstall"}<LoaderCircleIcon class="size-4 animate-spin" />{:else}<EraserIcon class="size-4" />{/if}
        </button>
      {/if}
    {/if}
    <button type="button" class="btn-icon preset-tonal" title="Release notes" aria-label="{app.name} release notes" disabled={!app.releaseNotes} onclick={() => onNotes(app)}>
      <ScrollTextIcon class="size-4" />
    </button>
  </footer>
</article>
