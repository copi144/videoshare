<script lang="ts">
  import { onMount } from 'svelte';
  import { createPlaylist, deletePlaylist, listCategories, listPlaylists, createTargetShareLink, listTargetShareLinks, deleteTargetShareLink } from '../lib/api';
  import MarkdownEditor from '../components/MarkdownEditor.svelte';

  export let onError: ((msg: string) => void) | undefined = undefined;

  interface Category {
    name: string;
    display_name: string;
    description?: string;
  }

  interface Playlist {
    name: string;
    description: string;
    category_name: string;
    playlist_type: string;
    created_at: string;
  }

  let playlists: Playlist[] = [];
  let categoryMap: Record<string, string> = {};
  let categories: Category[] = [];
  let formName = '';
  let formDisplayName = '';
  let formDescription = '';
  let formCategoryId = '';
  let formPlaylistType: 'video' | 'audio' | 'image' = 'video';
  let success: string | null = null;
  let loading = true;

  // Filters
  let selectedCategory = '';
  let selectedType = '';

  // Share link creation
  let showShareLink = false;
  let shareLinkPlaylistName = '';
  let shareLinkCategoryName = '';
  let shareLinkUrl = '';
  let shareLinkLoading = false;
  let shareLinkError: string | null = null;
  let shareLinkExpiry = 1440;
  let shareLinkExpiryMode: 'preset' | 'custom' = 'preset';
  let shareLinkCustomMinutes = 1440;
  $: if (shareLinkExpiryMode === 'custom') {
    shareLinkExpiry = shareLinkCustomMinutes;
  }

  // Name validation pattern
  const namePattern = /^[0-9A-Za-z\-]*$/;
  $: formNameValid = formName === '' || namePattern.test(formName);

  // Pagination (local state only)
  let limit = 50;
  let offset = 0;
  let total = 0;

  onMount(async () => {
    categories = (await listCategories()).categories;
    categoryMap = {};
    for (const cat of categories) {
      categoryMap[cat.name] = cat.display_name || cat.name;
    }
    await loadPlaylists();
  });

  async function loadPlaylists() {
    loading = true;
    try {
      const params: Record<string, string | number> = { limit, offset };
      if (selectedCategory) params.category_name = selectedCategory;
      if (selectedType) params.playlist_type = selectedType;
      const plData = await listPlaylists(params);
      playlists = plData.playlists;
      total = plData.total;
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to load playlists.';
      onError?.(msg);
    } finally {
      loading = false;
    }
  }

  function onFilterChange() {
    offset = 0;
    loadPlaylists();
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
      const result = await createPlaylist(formName.trim(), formDisplayName.trim(), formDescription.trim(), formCategoryId, formPlaylistType);
      if (result.ok) {
        success = `Playlist "${formName.trim()}" created successfully.`;
        formName = '';
        formDisplayName = '';
        formDescription = '';
        formCategoryId = '';
        formPlaylistType = 'video';
        await loadPlaylists();
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create playlist.';
      onError?.(msg);
    }
  }

  async function handleDelete(categoryName: string, name: string) {
    if (!confirm('Are you sure you want to delete this playlist? This action cannot be undone.')) {
      return;
    }
    try {
      await deletePlaylist(name, categoryName);
      await loadPlaylists();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to delete playlist.';
      onError?.(msg);
    }
  }

  function openShareLink(playlist: Playlist) {
    shareLinkPlaylistName = playlist.name;
    shareLinkCategoryName = playlist.category_name;
    shareLinkUrl = '';
    shareLinkError = null;
    shareLinkExpiry = 1440;
    shareLinkExpiryMode = 'preset';
    showShareLink = true;
    loadExistingShareLinks();
  }

  function closeShareLink() {
    showShareLink = false;
    shareLinkPlaylistName = '';
    shareLinkCategoryName = '';
    shareLinkUrl = '';
    shareLinkError = null;
  }

  async function handleCreateShareLink() {
    shareLinkLoading = true;
    shareLinkError = null;
    try {
      const result = await createTargetShareLink('playlist', shareLinkPlaylistName, shareLinkExpiry, shareLinkCategoryName);
      if (result.url) {
        shareLinkUrl = result.url;
      }
      // Reload existing links after creating a new one
      await loadExistingShareLinks();
    } catch (e: unknown) {
      shareLinkError = e instanceof Error ? e.message : 'Failed to create share link.';
    } finally {
      shareLinkLoading = false;
    }
  }

  // Existing share links management
  let existingShareLinks: Array<{id: string; expires_at: string | null; created_at: string}> = [];
  let loadingShareLinks = false;
  let linkToDelete: string | null = null;
  let deletingLink = false;

  async function loadExistingShareLinks() {
    if (!shareLinkPlaylistName) return;
    loadingShareLinks = true;
    try {
      const result = await listTargetShareLinks('playlist', shareLinkPlaylistName);
      existingShareLinks = result.share_links || [];
    } catch {
      existingShareLinks = [];
    } finally {
      loadingShareLinks = false;
    }
  }

  async function handleDeleteShareLink(linkId: string) {
    if (!confirm('Delete this share link? It will stop working immediately.')) return;
    deletingLink = true;
    linkToDelete = linkId;
    try {
      await deleteTargetShareLink(linkId);
      existingShareLinks = existingShareLinks.filter(l => l.id !== linkId);
    } catch (e: unknown) {
      shareLinkError = e instanceof Error ? e.message : 'Failed to delete share link.';
    } finally {
      deletingLink = false;
      linkToDelete = null;
    }
  }
</script>

<div class="space-y-4">
  {#if success}
    <div class="rounded-md bg-green-50 border border-green-200 px-3 py-2 text-sm text-green-700">{success}</div>
  {/if}

  <!-- Table -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Playlists</h2>

    <!-- Filter bar -->
    <div class="flex gap-3 mb-3">
      <select bind:value={selectedCategory} on:change={onFilterChange} class="min-w-[180px]">
        <option value="">All categories</option>
        {#each categories as cat}
          <option value={cat.name}>{cat.display_name || cat.name}{cat.name === 'global' ? ' (public)' : ''}</option>
        {/each}
      </select>
      <select bind:value={selectedType} on:change={onFilterChange} class="min-w-[180px]">
        <option value="">All types</option>
        <option value="video">Video</option>
        <option value="audio">Audio</option>
        <option value="image">Image</option>
      </select>
    </div>

    {#if loading}
      <p class="text-gray-500 text-sm">Loading playlists…</p>
    {:else if playlists.length === 0}
      <p class="text-gray-500 text-sm">
        {#if selectedCategory || selectedType}
          No playlists match the selected filters.
        {:else}
          No playlists yet.
        {/if}
      </p>
    {:else}
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200">
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Name</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Category</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Description</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Created</th>
            <th class="py-2 text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each playlists as pl}
            <tr class="border-b border-gray-100">
              <td class="py-2 pr-4"><a href="/#/l/{pl.category_name}/{pl.name}" class="text-indigo-600 hover:text-indigo-800 underline">{pl.name}</a></td>
              <td class="py-2 pr-4 text-gray-500">{categoryMap[pl.category_name] || 'Unknown'}</td>
              <td class="py-2 pr-4 text-gray-500">{pl.description || '—'}</td>
              <td class="py-2 pr-4 text-gray-500">{new Date(pl.created_at).toLocaleDateString()}</td>
              <td class="py-2">
                <div class="flex gap-1">
                  <button class="row-action-btn" type="button" on:click={() => openShareLink(pl)}>Share</button>
                  <button class="row-action-btn row-action-delete" type="button" on:click={() => handleDelete(pl.category_name, pl.name)}>Delete</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
      <div class="mt-3 flex items-center gap-2 text-sm">
        <span class="text-gray-500">{offset + 1}–{offset + playlists.length} of {total}</span>
        <button type="button" class="row-action-btn" on:click={() => { if (offset > 0) { offset = Math.max(0, offset - limit); loadPlaylists(); } }} disabled={offset === 0}>Prev</button>
        <button type="button" class="row-action-btn" on:click={() => { offset += limit; loadPlaylists(); }} disabled={offset + playlists.length >= total}>Next</button>
      </div>
    {/if}
  </div>

  <!-- Create form -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Create Playlist</h2>
    <form on:submit|preventDefault={handleCreate}>
      <!-- Name + Display Name inline -->
      <div class="flex gap-3 mb-3">
        <div class="flex-1">
          <label for="name" class="block text-sm font-medium text-gray-700 mb-1">Name</label>
          <input type="text" id="name" name="name" bind:value={formName} required pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" class="w-full" class:border-red-500={formName !== '' && !formNameValid} />
        </div>
        <div class="flex-1">
          <label for="display_name" class="block text-sm font-medium text-gray-700 mb-1">Display Name</label>
          <input type="text" id="display_name" bind:value={formDisplayName} class="w-full" />
        </div>
      </div>
      <div class="mb-3">
        <label class="block text-sm font-medium text-gray-700 mb-1">Description (Markdown)</label>
        <MarkdownEditor bind:value={formDescription} placeholder="Optional markdown description..." />
      </div>
      <div class="flex gap-3 mb-3">
        <div class="flex-1">
          <label for="category" class="block text-sm font-medium text-gray-700 mb-1">Category</label>
          <select id="category" name="category_id" bind:value={formCategoryId} required class="w-full">
            <option value="">— Select —</option>
            {#each categories as cat}
              <option value={cat.name}>{cat.display_name || cat.name}{cat.name === 'global' ? ' (public)' : ''}</option>
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
      <div class="flex justify-end">
        <button type="submit" class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700">Create</button>
      </div>
    </form>
  </div>
</div>

{#if showShareLink}
  <div class="fixed inset-0 bg-black/30 z-40" on:click={closeShareLink}></div>
  <div class="fixed inset-0 flex items-center justify-center z-50">
    <div class="rounded-lg border border-gray-200 bg-white p-6 max-w-md w-full shadow-xl mx-4">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-base font-semibold text-gray-900">Share Link: {shareLinkPlaylistName}</h3>
        <button class="text-gray-400 hover:text-gray-600 text-xl leading-none" on:click={closeShareLink}>&times;</button>
      </div>

      {#if shareLinkUrl}
        <div class="space-y-3">
          <div class="rounded-md bg-green-50 border border-green-200 px-3 py-2 text-sm text-green-700">
            Link created!
          </div>
          <div class="text-sm text-gray-500 break-all">{window.location.origin}{shareLinkUrl}</div>
          <input type="text" readonly value={window.location.origin + shareLinkUrl} class="w-full text-sm" on:focus={(e) => e.currentTarget.select()} />
          <div class="flex justify-end border-t border-gray-100 pt-3">
            <button
              class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50"
              type="button"
              on:click={closeShareLink}
            >Close</button>
          </div>
        </div>
      {:else}
        <!-- Existing share links -->
        {#if existingShareLinks.length > 0}
          <div class="mb-3 border border-gray-200 rounded-md p-3">
            <h4 class="text-sm font-medium text-gray-700 mb-2">Existing Share Links</h4>
            <div class="space-y-2 max-h-40 overflow-y-auto">
              {#each existingShareLinks as link}
                <div class="flex items-center justify-between text-sm bg-gray-50 rounded px-2 py-1.5">
                  <div class="flex-1 min-w-0">
                    <p class="text-xs text-gray-500 truncate font-mono">{link.id}</p>
                    {#if link.expires_at}
                      <p class="text-xs text-gray-400">Expires: {new Date(link.expires_at).toLocaleString()}</p>
                    {:else}
                      <p class="text-xs text-gray-400">No expiry</p>
                    {/if}
                  </div>
                  <button type="button" disabled={deletingLink && linkToDelete === link.id}
                    class="text-red-600 hover:text-red-800 text-xs ml-2 disabled:opacity-50"
                    on:click={() => handleDeleteShareLink(link.id)}>
                    {deletingLink && linkToDelete === link.id ? '…' : 'Delete'}
                  </button>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <div class="space-y-3">
          <div class="rounded-md bg-blue-50 border border-blue-200 px-3 py-2 text-sm text-blue-700">
            Anyone with this link can access all videos in this playlist.
          </div>

          <label class="block text-sm font-medium text-gray-700">Expires after</label>
          <div class="flex gap-4 mb-2">
            <label class="inline-flex items-center gap-1.5 text-sm">
              <input type="radio" name="slExpiryMode" value="preset" bind:group={shareLinkExpiryMode} />
              Preset
            </label>
            <label class="inline-flex items-center gap-1.5 text-sm">
              <input type="radio" name="slExpiryMode" value="custom" bind:group={shareLinkExpiryMode} />
              Custom
            </label>
          </div>

          {#if shareLinkExpiryMode === 'preset'}
            <select bind:value={shareLinkExpiry} class="w-full">
              <option value={1}>1 minute</option>
              <option value={5}>5 minutes</option>
              <option value={30}>30 minutes</option>
              <option value={60}>1 hour</option>
              <option value={360}>6 hours</option>
              <option value={720}>12 hours</option>
              <option value={1440}>1 day</option>
              <option value={4320}>3 days</option>
              <option value={10080}>7 days</option>
              <option value={43200}>30 days</option>
              <option value={129600}>90 days</option>
              <option value={525600}>365 days</option>
            </select>
          {:else}
            <div>
              <input type="number" bind:value={shareLinkCustomMinutes} min={1} max={525600} class="w-full" />
              <p class="text-xs text-gray-500 mt-1">Minutes (1–525600)</p>
            </div>
          {/if}

          {#if shareLinkError}
            <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{shareLinkError}</div>
          {/if}

          <div class="flex justify-end gap-2 border-t border-gray-100 pt-3">
            <button
              class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50"
              type="button"
              on:click={closeShareLink}
            >Cancel</button>
            <button
              class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
              type="button"
              disabled={shareLinkLoading}
              on:click={handleCreateShareLink}
            >
              {shareLinkLoading ? 'Creating…' : 'Create Link'}
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .row-action-btn {
    padding: 0.2rem 0.5rem;
    font-size: 0.75rem;
    border: 1px solid #d1d5db;
    border-radius: 0.25rem;
    background: white;
    color: #374151;
    cursor: pointer;
    white-space: nowrap;
  }
  .row-action-btn:hover {
    background: #f3f4f6;
  }
  .row-action-delete {
    color: #dc2626;
    border-color: #fecaca;
  }
  .row-action-delete:hover {
    background: #fef2f2;
  }
</style>
