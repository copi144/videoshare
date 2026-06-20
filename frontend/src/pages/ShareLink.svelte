<script lang="ts">
  import { onMount } from 'svelte';
  import { authenticateShareLink } from '../lib/api';

  export let id: string;
  export let password: string;

  let error: string | null = null;
  let loading = true;

  onMount(async () => {
    if (!id || !password) {
      error = 'Invalid share link.';
      loading = false;
      return;
    }

    try {
      const result = await authenticateShareLink(id, password);
      if (result.ok && result.redirect) {
        window.location.hash = result.redirect;
      } else {
        error = 'Invalid or expired share link.';
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Authentication failed.';
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <p aria-busy="true">Authenticating share link…</p>
{:else if error}
  <div class="rounded-lg border border-gray-200 bg-white p-6 max-w-md mx-auto mt-8">
    <h2 class="text-lg font-semibold text-gray-900 mb-2">Access Denied</h2>
    <p class="text-sm text-gray-600">{error}</p>
  </div>
{/if}
