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
      const result = await createPlaylist(formName.trim(), formDescription.trim(), formCategoryId);
      if (result.ok) {
        success = `Playlist "${formName.trim()}" created successfully.`;
        formName = '';
        formDescription = '';
        formCategoryId = '';
        await loadData();
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create playlist.';
      onError?.(msg);
    }
  }
</script>

<h1>Playlist Management</h1>

{#if success}
  <article>{success}</article>
{/if}

<article>
  <h2>Create Playlist</h2>
  <form on:submit|preventDefault={handleCreate}>
    <label for="name">
      Name
      <input type="text" id="name" name="name" bind:value={formName} required pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" />
    </label>
    <label for="description">
      Description
      <textarea id="description" name="description" bind:value={formDescription}></textarea>
    </label>
    <label for="category">
      Category
      <select id="category" name="category_id" bind:value={formCategoryId} required>
        <option value="">— Select a category —</option>
        {#each categories as cat}
          <option value={cat.id}>
            {cat.name}{cat.id === 'global' ? ' (public)' : ''}
          </option>
        {/each}
      </select>
    </label>
    <button type="submit">Create</button>
  </form>
</article>

<h2>Existing Playlists</h2>
{#if loading}
  <p aria-busy="true">Loading playlists…</p>
{:else if playlists.length === 0}
  <p>No playlists yet. Create one above.</p>
{:else}
  <table role="grid">
    <thead>
      <tr>
        <th>Name</th>
        <th>Category</th>
        <th>Description</th>
        <th>Created</th>
      </tr>
    </thead>
    <tbody>
      {#each playlists as pl}
        <tr>
          <td>{pl.name}</td>
          <td>{categoryMap[pl.category_id] || 'Unknown'}</td>
          <td>{pl.description || '—'}</td>
          <td>{new Date(pl.created_at).toLocaleDateString()}</td>
        </tr>
      {/each}
    </tbody>
  </table>

  <!-- Pagination -->
  <div style="margin-top: 0.5rem; display: flex; align-items: center; gap: 0.5rem;">
    <span>{playlists.length > 0 ? offset + 1 : 0}–{offset + playlists.length} of {total}</span>
    <button type="button" on:click={() => { if (offset > 0) { offset = Math.max(0, offset - limit); loadData(); } }} disabled={offset === 0}>Prev</button>
    <button type="button" on:click={() => { offset += limit; loadData(); }} disabled={offset + playlists.length >= total}>Next</button>
  </div>
{/if}
