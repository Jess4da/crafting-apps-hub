<script lang="ts">
  import "./layout.css";
  import { Toast } from "@skeletonlabs/skeleton-svelte";
  import { toaster } from "#lib/hub.svelte.ts";

  let { children } = $props();
</script>

{@render children()}

<Toast.Group {toaster}>
  {#snippet children(toast)}
    <Toast {toast} class="glass--float note">
      <Toast.Message>
        <Toast.Title class="note__title">{toast.title}</Toast.Title>
        <Toast.Description class="note__body">{toast.description}</Toast.Description>
      </Toast.Message>
      <Toast.CloseTrigger />
    </Toast>
  {/snippet}
</Toast.Group>

<style>
  /* Toasts only report failures: floating glass with a danger ring, no border. */
  :global(.glass--float.note) {
    border-radius: 18px;
    color: var(--color-ink);
    box-shadow:
      0 0 0 1px var(--color-danger),
      0 1.5rem 3rem -1.5rem var(--glass-shadow),
      inset 0 1px 0 var(--glass-inner);
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
