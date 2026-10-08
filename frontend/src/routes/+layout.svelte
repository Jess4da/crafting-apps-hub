<script lang="ts">
  import "./layout.css";
  import { Toast } from "@skeletonlabs/skeleton-svelte";
  import { toaster } from "#lib/hub.svelte.ts";

  let { children } = $props();
</script>

{@render children()}

<Toast.Group {toaster}>
  {#snippet children(toast)}
    <Toast {toast}>
      <Toast.Message>
        <Toast.Title>{toast.title}</Toast.Title>
        <Toast.Description>{toast.description}</Toast.Description>
      </Toast.Message>
      <Toast.CloseTrigger />
    </Toast>
  {/snippet}
</Toast.Group>
