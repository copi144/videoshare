<script lang="ts">
  import { onMount } from 'svelte';
  import { authenticateShareLink } from '../lib/api';
  import MainApp from './MainApp.svelte';

  export let id: string;
  export let password: string;

  let loading = true;
  let error: string | null = null;
  let targetType: string | null = null;
  let targetId: string | null = null;
  let targetName: string | null = null;

  onMount(async () => {
    try {
      const result = await authenticateShareLink(id, password);
      targetType = result.target_type;
      targetId = result.target_id;
      targetName = result.target_name;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Invalid or expired share link.';
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <div class="max-w-4xl mx-auto mt-8 px-4">
    <p class="text-gray-500 text-sm">Loading shared content…</p>
  </div>
{:else if error}
  <div class="max-w-4xl mx-auto mt-8 px-4">
    <div class="rounded-lg border border-red-200 bg-red-50 p-6">
      <h2 class="text-lg font-semibold text-red-800 mb-2">Access Denied</h2>
      <p class="text-sm text-red-600">{error}</p>
      <p class="text-xs text-red-500 mt-2">The share link may be invalid, expired, or the password is incorrect.</p>
    </div>
  </div>
{:else}
  <MainApp
    sharedMode={true}
    shareTargetType={targetType}
    shareTargetId={targetId}
    shareTargetName={targetName}
    shareLinkId={id}
    shareLinkPassword={password}
  />
{/if}
