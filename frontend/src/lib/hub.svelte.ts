import { Events } from "@wailsio/runtime";
import { createToaster } from "@skeletonlabs/skeleton-svelte";
import { HubService } from "#bindings/craftinghub/index.ts";
import type { AppState, Progress, Settings } from "#bindings/craftinghub/internal/hub/models.ts";

export const toaster = createToaster({ placement: "bottom-end" });

/** Reactive store shared by every component. Backend is the source of truth. */
class HubStore {
  apps = $state<AppState[]>([]);
  settings = $state<Settings>({ installRoot: "" });
  platform = $state("");
  refreshing = $state(false);
  progress = $state<Record<string, Progress>>({});
  /** ids with a running install/update/uninstall */
  busy = $state<Record<string, "install" | "uninstall">>({});

  installed = $derived(this.apps.filter((a) => a.installed));
  updates = $derived(this.apps.filter((a) => a.updateAvailable));

  async init() {
    Events.On("hub:progress", (e) => {
      this.progress[e.data.id] = e.data;
    });
    [this.apps, this.settings, this.platform] = await Promise.all([
      HubService.List(),
      HubService.GetSettings(),
      HubService.Platform(),
    ]);
    await this.refresh();
  }

  async refresh() {
    this.refreshing = true;
    try {
      this.apps = await HubService.Refresh();
      const failed = this.apps.find((a) => a.error);
      if (failed) toaster.warning({ title: "Couldn't check some apps", description: failed.error });
    } catch (e) {
      toaster.error({ title: "Refresh failed", description: String(e) });
    } finally {
      this.refreshing = false;
    }
  }

  async install(app: AppState) {
    const verb = app.installed ? "Updated" : "Installed";
    this.busy[app.id] = "install";
    try {
      const s = await HubService.Install(app.id);
      this.apps = this.apps.map((a) => (a.id === s.id ? s : a));
      toaster.success({ title: `${verb} ${s.name}`, description: `Version ${s.installed?.version}` });
    } catch (e) {
      toaster.error({ title: `${app.name} failed`, description: errText(e) });
    } finally {
      delete this.busy[app.id];
      delete this.progress[app.id];
    }
  }

  async updateAll() {
    // Sequential: keeps bandwidth and GitHub rate limits sane.
    for (const a of this.updates) await this.install(a);
  }

  async uninstall(app: AppState, clean: boolean) {
    this.busy[app.id] = "uninstall";
    try {
      const s = await HubService.Uninstall(app.id, clean);
      this.apps = this.apps.map((a) => (a.id === s.id ? s : a));
      toaster.info({ title: clean ? `Removed ${app.name} and its settings` : `Removed ${app.name}` });
    } catch (e) {
      toaster.error({ title: `Couldn't fully remove ${app.name}`, description: errText(e) });
      this.apps = await HubService.List(); // a partial removal may have happened
    } finally {
      delete this.busy[app.id];
    }
  }

  async launch(app: AppState) {
    try {
      await HubService.Launch(app.id);
    } catch (e) {
      toaster.error({ title: `Couldn't open ${app.name}`, description: errText(e) });
    }
  }

  async openFolder(id: string) {
    try {
      await HubService.OpenInstallDir(id);
    } catch (e) {
      toaster.error({ title: "Couldn't open folder", description: errText(e) });
    }
  }

  async chooseInstallRoot() {
    try {
      this.settings = await HubService.ChooseInstallRoot();
    } catch (e) {
      toaster.error({ title: "Couldn't change folder", description: errText(e) });
    }
  }
}

/** Wails rejects with an Error-like object whose message is the Go error. */
function errText(e: unknown): string {
  return e instanceof Error ? e.message : typeof e === "object" && e && "message" in e ? String(e.message) : String(e);
}

export function formatBytes(n: number): string {
  if (!n) return "";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(n) / Math.log(1024)), units.length - 1);
  return `${(n / 1024 ** i).toFixed(i ? 1 : 0)} ${units[i]}`;
}

export const hub = new HubStore();
