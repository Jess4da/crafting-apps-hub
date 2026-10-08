<script lang="ts">
  import "./layout.css";
  import { Toast } from "@skeletonlabs/skeleton-svelte";
  import { toaster } from "#lib/hub.svelte.ts";

  let { children } = $props();
</script>

{@render children()}

<Toast.Group {toaster}>
  {#snippet children(toast)}
    <Toast {toast} class="note">
      <Toast.Message>
        <Toast.Title class="note__title">{toast.title}</Toast.Title>
        <Toast.Description class="note__body">{toast.description}</Toast.Description>
      </Toast.Message>
      <Toast.CloseTrigger />
    </Toast>
  {/snippet}
</Toast.Group>

<style>
  /* Toasts only report failures: paper sheet, danger hairline, no fill. */
  :global(.note) {
    border: 1px solid var(--color-danger);
    border-radius: 0.5rem;
    background: var(--color-paper);
    color: var(--color-ink);
    box-shadow: 0 1px 2px var(--color-shadow);
  }
  :global(.note__title) {
    font-weight: 600;
    color: var(--color-danger);
  }
  :global(.note__body) {
    color: var(--color-muted);
    font-size: 0.875rem;
    overflow-wrap: anywhere;
  }
</style>
