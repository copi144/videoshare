<script lang="ts">
  import { onMount } from 'svelte';
  import { createSession } from '../lib/api';

  export let id: string;
  export let password: string = '';
  export let onSuccess: (id: string, password?: string) => void;

  let localPassword = '';
  let error: string | null = null;
  let loading = true;
  let needsShareLink = false;

  onMount(async () => {
    if (!id) {
      error = 'Invalid video ID.';
      loading = false;
      return;
    }

    // If a password was provided (e.g. from a share link URL), validate it
    if (password) {
      try {
        const result = await createSession('share', { resource_id: id, password });
        if (result.ok) {
          onSuccess(id, password);
          return;
        }
        error = 'Invalid or expired share link.';
      } catch (e: unknown) {
        error = e instanceof Error ? e.message : 'Authentication failed.';
      } finally {
        loading = false;
      }
      return;
    }

    // No password — try auto-auth (global category or user with category access)
    try {
      const result = await createSession('share', { resource_id: id, password: '' });
      if (result.ok) {
        onSuccess(id, '');
        return;
      }
      // Auto-auth failed — show share link password form
      needsShareLink = true;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to access video.';
    } finally {
      loading = false;
    }
  });

  async function handleSubmit() {
    error = null;
    loading = true;
    try {
      const result = await createSession('share', { resource_id: id, password: localPassword });
      if (result.ok) {
        onSuccess(id, localPassword);
      } else {
        error = 'Invalid or expired share link.';
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Authentication failed.';
    } finally {
      loading = false;
    }
  }
</script>

{#if loading}
  <p aria-busy="true">Checking video access…</p>
{:else if needsShareLink}
  <div class="rounded-lg border border-gray-200 bg-white p-6 max-w-md mx-auto mt-8">
    <h2 class="text-lg font-semibold text-gray-900 mb-2">Share Link Password Required</h2>
    <p class="text-sm text-gray-600 mb-4">This video requires a share link password to access. If you have a share link, enter the password below.</p>
    <form on:submit|preventDefault={handleSubmit}>
      <div class="mb-3">
        <input type="password" id="share-password" name="password" bind:value={localPassword} placeholder="Enter share link password" required autocomplete="off" class="w-full px-3 py-2 border border-gray-300 rounded-md text-sm" />
      </div>
      {#if error}
        <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700 mb-3">{error}</div>
      {/if}
      <button type="submit" disabled={loading} aria-busy={loading} class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
        {loading ? 'Authenticating…' : 'Access Video'}
      </button>
    </form>
  </div>
{:else if error}
  <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700 max-w-md mx-auto mt-8">{error}</div>
{/if}
