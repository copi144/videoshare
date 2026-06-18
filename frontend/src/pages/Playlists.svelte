<script lang="ts">
  import { onMount } from 'svelte';
  import { isAuthenticated, navigate } from '../stores/auth';
  import { createPlaylist, listCategories, listPlaylists } from '../lib/api';

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
  let error: string | null = null;
  let success: string | null = null;
  let loading = true;

  onMount(async () => {
    if (!$isAuthenticated) {
      navigate('/login');
      return;
    }
    await loadData();
  });

  async function loadData() {
    error = null;
    loading = true;
    try {
      const [catData, plData] = await Promise.all([
        listCategories(),
        listPlaylists(),
      ]);
      categories = catData.categories;
      categoryMap = {};
      for (const cat of catData.categories) {
        categoryMap[cat.id] = cat.name;
      }
      playlists = plData.playlists;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load data.';
    } finally {
      loading = false;
    }
  }

  async function handleCreate() {
    error = null;
    success = null;

    if (!formName.trim()) {
      error = 'Playlist name is required.';
      return;
    }
    if (!formCategoryId) {
      error = 'Please select a category.';
      return;
    }

    try {
      const result = await createPlaylist(formName.trim(), formDescription.trim(), formCategoryId);
      if (result.ok) {
        success = `Playlist "${formName.trim()}" created successfully.`;
        formName = '';
        formDescription = '';
        formCategoryId = '';
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to create playlist.';
    }
  }
</script>

<h1>Playlist Management</h1>

{#if error}
  <article class="error-box">{error}</article>
{/if}
{#if success}
  <article>{success}</article>
{/if}

<article>
  <h2>Create Playlist</h2>
  <form on:submit|preventDefault={handleCreate}>
    <label for="name">
      Name
      <input type="text" id="name" name="name" bind:value={formName} required />
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
            {cat.name}{cat.id.startsWith('00000000') ? ' (public)' : ''}
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
{/if}
