<script lang="ts">
  import { onMount } from 'svelte';
  import { createPlaylist, listCategories, listPlaylists } from '../lib/api';

  export let onError: ((msg: string) => void) | undefined = undefined;

  interface Category {
    id: string;
    name: string;
    description?: string;
  }

  interface Playlist {
    id: string;
    name: string;
    description: string;
    category_id: string;
    created_at: string;
  }

  let playlists: Playlist[] = [];
  let categoryMap: Record<string, string> = {};
  let categories: Category[] = [];
  let formName = '';
  let formDescription = '';
  let formCategoryId = '';
  let formPlaylistType: 'video' | 'audio' | 'image' = 'video';
  let success: string | null = null;
  let loading = true;

  // Pagination (local state only)
  let limit = 50;
  let offset = 0;
  let total = 0;

  onMount(async () => {
    await loadData();
  });

  async function loadData() {
    loading = true;
    try {
      const [catData, plData] = await Promise.all([
        listCategories(),
        listPlaylists({ limit, offset }),
      ]);
      categories = catData.categories;
      categoryMap = {};
      for (const cat of catData.categories) {
        categoryMap[cat.id] = cat.name;
      }
      playlists = plData.playlists;
      total = plData.total;
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to load data.';
      onError?.(msg);
    } finally {
      loading = false;
    }
  }

  async function handleCreate() {
    success = null;

    if (!formName.trim()) {
      onError?.('Playlist name is required.');
      return;
    }
    if (!formCategoryId) {
      onError?.('Please select a category.');
      return;
    }

    try {
      const result = await createPlaylist(formName.trim(), formDescription.trim(), formCategoryId, formPlaylistType);
      if (result.ok) {
        success = `Playlist "${formName.trim()}" created successfully.`;
        formName = '';
        formDescription = '';
        formCategoryId = '';
        formPlaylistType = 'video';
        await loadData();
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create playlist.';
      onError?.(msg);
    }
  }
</script>

<div class="space-y-4">
  {#if success}
    <div class="rounded-md bg-green-50 border border-green-200 px-3 py-2 text-sm text-green-700">{success}</div>
  {/if}

  <!-- Create form -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Create Playlist</h2>
    <form on:submit|preventDefault={handleCreate} class="space-y-3">
      <div>
        <label for="name" class="block text-sm font-medium text-gray-700 mb-1">Name</label>
        <input type="text" id="name" name="name" bind:value={formName} required pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" class="w-full" />
      </div>
      <div>
        <label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea id="description" name="description" bind:value={formDescription} class="w-full"></textarea>
      </div>
      <div class="flex gap-3">
        <div class="flex-1">
          <label for="category" class="block text-sm font-medium text-gray-700 mb-1">Category</label>
          <select id="category" name="category_id" bind:value={formCategoryId} required class="w-full">
            <option value="">— Select —</option>
            {#each categories as cat}
              <option value={cat.id}>{cat.name}{cat.id === 'global' ? ' (public)' : ''}</option>
            {/each}
          </select>
        </div>
        <div class="flex-1">
          <label for="playlist_type" class="block text-sm font-medium text-gray-700 mb-1">Type</label>
          <select id="playlist_type" name="playlist_type" bind:value={formPlaylistType} class="w-full">
            <option value="video">Video</option>
            <option value="audio">Audio</option>
            <option value="image">Image</option>
          </select>
        </div>
      </div>
      <button type="submit" class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700">Create</button>
    </form>
  </div>

  <!-- Table -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Playlists</h2>
    {#if loading}
      <p class="text-gray-500 text-sm">Loading playlists…</p>
    {:else if playlists.length === 0}
      <p class="text-gray-500 text-sm">No playlists yet. Create one above.</p>
    {:else}
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200">
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Name</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Category</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Description</th>
            <th class="py-2 text-xs font-medium text-gray-500 uppercase">Created</th>
          </tr>
        </thead>
        <tbody>
          {#each playlists as pl}
            <tr class="border-b border-gray-100">
              <td class="py-2 pr-4">{pl.name}</td>
              <td class="py-2 pr-4 text-gray-500">{categoryMap[pl.category_id] || 'Unknown'}</td>
              <td class="py-2 pr-4 text-gray-500">{pl.description || '—'}</td>
              <td class="py-2 text-gray-500">{new Date(pl.created_at).toLocaleDateString()}</td>
            </tr>
          {/each}
        </tbody>
      </table>
      <div class="mt-3 flex items-center gap-2 text-sm">
        <span class="text-gray-500">{offset + 1}–{offset + playlists.length} of {total}</span>
        <button type="button" class="row-action-btn" on:click={() => { if (offset > 0) { offset = Math.max(0, offset - limit); loadData(); } }} disabled={offset === 0}>Prev</button>
        <button type="button" class="row-action-btn" on:click={() => { offset += limit; loadData(); }} disabled={offset + playlists.length >= total}>Next</button>
      </div>
    {/if}
  </div>
</div>
