<script lang="ts">
  import { onMount } from 'svelte';
  import { createSession, getResource } from '../lib/api';

  export let id: string;
  export let password: string = '';
  export let onSuccess: (id: string, password?: string) => void;

  let localPassword = password;
  let error: string | null = null;
  let loading = true;
  let needsPassword = false;

  onMount(async () => {
    if (!id) {
      error = 'Invalid video ID.';
      loading = false;
      return;
    }

    // If a password was provided (e.g. from a share link URL), use it directly
    if (password) {
      try {
        const result = await createSession('share', { resource_id: id, password });
        if (result.ok) {
          onSuccess(id, password);
          return;
        }
        error = 'Failed to access video with provided password.';
      } catch (e: unknown) {
        error = e instanceof Error ? e.message : 'Authentication failed.';
      } finally {
        loading = false;
      }
      return;
    }

    try {
      const resource = await getResource(id);
      if (resource.category_id === 'global') {
        // Public video — auto-authenticate and redirect
        const result = await createSession('share', { resource_id: id, password: '' });
        if (result.ok) {
          onSuccess(id, '');
          return;
        }
        error = 'Failed to access video.';
      } else {
        // Password-protected — show the form
        needsPassword = true;
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load video info.';
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
        error = 'Incorrect password.';
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
{:else if needsPassword}
  <h1>Enter Video Password</h1>
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <form on:submit|preventDefault={handleSubmit}>
      <label for="password">
        Password
        <input type="password" id="password" name="password" bind:value={localPassword} required autocomplete="off" />
      </label>
      {#if error}
        <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{error}</div>
      {/if}
      <button type="submit" disabled={loading} aria-busy={loading}>
        {loading ? 'Authenticating…' : 'Access Video'}
      </button>
    </form>
  </div>
{:else if error}
  <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{error}</div>
{/if}
