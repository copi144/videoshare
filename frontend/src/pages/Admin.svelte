<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { isAuthenticated, navigate } from '../stores/auth';
  import { listResources, uploadVideo, deleteResource, listCategories } from '../lib/api';
  import {
    getWatchHistory,
    clearWatchHistory,
    removeWatchEntry,
    addSearchHistory,
    getSearchHistory,
    clearSearchHistory,
  } from '../stores/history';

  interface Resource {
    id: string;
    title: string;
    description: string;
    content_type: string;
    file_size: number;
    views: number;
    created_at: string;
    uploaded_by: string;
    uploaded_username: string;
    category_id: string;
    category_name: string;
    password_hash: string;
  }

  interface Category {
    id: string;
    name: string;
    description?: string;
  }

  let resources: Resource[] = [];
  let categories: Category[] = [];
  let uploadForm = { title: '', description: '', category_id: '', password: '' };
  let selectedFile: File | null = null;
  let error: string | null = null;
  let uploadError: string | null = null;
  let loading = true;
  let uploading = false;
  let copySuccess: string | null = null;

  // Search state
  let searchQuery = '';
  let searchInput = '';
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let showRecentSearches = false;
  let searchFocused = false;

  // Watch history
  let watchHistory = getWatchHistory();

  function onFileChange(e: Event) {
    selectedFile = (e.target as HTMLInputElement).files?.[0] ?? null;
  }

  $: selectedCategory = categories.find(c => c.id === uploadForm.category_id);
  $: isGlobal = selectedCategory ? selectedCategory.id.startsWith('00000000') : false;

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  function formatRelativeTime(isoString: string): string {
    const now = Date.now();
    const then = new Date(isoString).getTime();
    const diffMs = now - then;
    if (diffMs < 0) return 'just now';

    const seconds = Math.floor(diffMs / 1000);
    if (seconds < 60) return 'just now';

    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;

    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;

    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;

    const months = Math.floor(days / 30);
    if (months < 12) return `${months}mo ago`;

    const years = Math.floor(months / 12);
    return `${years}y ago`;
  }

  $: watchHistory = getWatchHistory();

  // Search filtering with debounce
  $: filteredResources = resources.filter((res) => {
    if (!searchQuery.trim()) return true;
    return res.title.toLowerCase().includes(searchQuery.trim().toLowerCase());
  });

  $: recentSearches = getSearchHistory();

  function onSearchInput() {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      searchQuery = searchInput;
      if (searchInput.trim()) {
        addSearchHistory(searchInput.trim());
        recentSearches = getSearchHistory();
      }
    }, 300);
  }

  function onSearchFocus() {
    searchFocused = true;
    recentSearches = getSearchHistory();
    if (recentSearches.length > 0) {
      showRecentSearches = true;
    }
  }

  function onSearchBlur() {
    // Delay so click on suggestion registers
    setTimeout(() => {
      searchFocused = false;
      showRecentSearches = false;
    }, 200);
  }

  function selectRecentSearch(query: string) {
    searchInput = query;
    searchQuery = query;
    showRecentSearches = false;
  }

  function clearWatchEntry(id: string) {
    removeWatchEntry(id);
    watchHistory = getWatchHistory();
  }

  function handleClearHistory() {
    clearWatchHistory();
    watchHistory = getWatchHistory();
  }

  async function loadData() {
    error = null;
    try {
      const [resData, catData] = await Promise.all([
        listResources(),
        listCategories(),
      ]);
      resources = resData.resources;
      categories = catData.categories;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load data.';
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    if (!$isAuthenticated) {
      navigate('/login');
      return;
    }
    await loadData();
  });

  onDestroy(() => {
    if (debounceTimer) clearTimeout(debounceTimer);
  });

  async function handleUpload() {
    uploadError = null;
    if (!selectedFile) {
      uploadError = 'Please select a video file.';
      return;
    }
    if (!uploadForm.title.trim()) {
      uploadError = 'Title is required.';
      return;
    }
    if (!uploadForm.category_id) {
      uploadError = 'Please select a category.';
      return;
    }
    if (!isGlobal && !uploadForm.password.trim()) {
      uploadError = 'Password is required for non-public categories.';
      return;
    }

    uploading = true;
    try {
      const fd = new FormData();
      fd.append('file', selectedFile);
      fd.append('title', uploadForm.title.trim());
      fd.append('description', uploadForm.description.trim());
      fd.append('category_id', uploadForm.category_id);
      if (uploadForm.password.trim()) {
        fd.append('password', uploadForm.password.trim());
      }
      await uploadVideo(fd);
      // Reset form
      uploadForm = { title: '', description: '', category_id: '', password: '' };
      selectedFile = null;
      // Reload resources
      await loadData();
    } catch (e: unknown) {
      uploadError = e instanceof Error ? e.message : 'Upload failed.';
    } finally {
      uploading = false;
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Are you sure you want to delete this video? This action cannot be undone.')) {
      return;
    }
    try {
      await deleteResource(id);
      resources = resources.filter(r => r.id !== id);
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to delete resource.';
    }
  }

  function copyShareLink(id: string) {
    const url = `${window.location.origin}/#/s/${id}`;
    navigator.clipboard.writeText(url).then(() => {
      copySuccess = id;
      setTimeout(() => { copySuccess = null; }, 2000);
    }).catch(() => {
      // Fallback for older browsers
      const ta = document.createElement('textarea');
      ta.value = url;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
      copySuccess = id;
      setTimeout(() => { copySuccess = null; }, 2000);
    });
  }
</script>

<h1>Video Management</h1>

{#if error}
  <article class="error-box">{error}</article>
{/if}

<!-- Search Bar -->
<div class="search-container" style="position: relative; margin-bottom: 1rem;">
  <input
    type="search"
    placeholder="Search videos by title…"
    bind:value={searchInput}
    on:input={onSearchInput}
    on:focus={onSearchFocus}
    on:blur={onSearchBlur}
    style="width: 100%;"
  />
  {#if showRecentSearches && recentSearches.length > 0}
    <div class="recent-searches" style="position: absolute; top: 100%; left: 0; right: 0; z-index: 10; background: var(--card-background-color, #fff); border: 1px solid var(--card-border-color, #ccc); border-radius: 0 0 var(--border-radius, 4px) var(--border-radius, 4px); box-shadow: 0 4px 12px rgba(0,0,0,0.15);">
      <div style="display: flex; justify-content: space-between; align-items: center; padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--card-border-color, #eee);">
        <small style="font-weight: 600;">Recent searches</small>
        <button class="outline secondary" style="padding: 0.15rem 0.5rem; font-size: 0.75rem;" on:click|stopPropagation={clearSearchHistory}>Clear</button>
      </div>
      {#each recentSearches as sr}
        <button
          class="outline secondary"
          style="display: block; width: 100%; text-align: left; border: none; border-radius: 0; padding: 0.5rem 0.75rem; font-size: 0.875rem; cursor: pointer;"
          on:click|stopPropagation={() => selectRecentSearch(sr.query)}
          on:mousedown|preventDefault
        >
          {sr.query}
        </button>
      {/each}
    </div>
  {/if}
</div>

<!-- Upload Form -->
<article>
  <h2>Upload Video</h2>
  <form on:submit|preventDefault={handleUpload}>
    <label for="title">
      Title
      <input type="text" id="title" name="title" bind:value={uploadForm.title} required />
    </label>
    <label for="description">
      Description
      <textarea id="description" name="description" bind:value={uploadForm.description}></textarea>
    </label>
    <label for="category">
      Category
      <select id="category" name="category_id" bind:value={uploadForm.category_id} required>
        <option value="">— Select a category —</option>
        {#each categories as cat}
          <option value={cat.id}>
            {cat.name}{cat.id.startsWith('00000000') ? ' (public)' : ''}
          </option>
        {/each}
      </select>
    </label>
    <label for="file">
      Video File
      <input
        type="file"
        id="file"
        name="file"
        accept="video/mp4,video/webm,video/x-matroska"
        on:change={onFileChange}
        required
      />
    </label>
    {#if !isGlobal && uploadForm.category_id}
      <label for="password">
        Password (required for this category)
        <input type="text" id="password" name="password" bind:value={uploadForm.password} />
      </label>
    {/if}
    {#if uploadError}
      <article class="error-box">{uploadError}</article>
    {/if}
    <button type="submit" disabled={uploading} aria-busy={uploading}>
      {uploading ? 'Uploading…' : 'Upload'}
    </button>
  </form>
</article>

<!-- Resources Table -->
<h2>My Videos</h2>
{#if loading}
  <p aria-busy="true">Loading videos…</p>
{:else}
  <figure>
    <table role="grid">
      <thead>
        <tr>
          <th>Title</th>
          <th>Category</th>
          <th>Views</th>
          <th>Size</th>
          <th>Share Link</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each filteredResources as res}
          <tr>
            <td>{res.title}</td>
            <td>{res.category_name}</td>
            <td>{res.views}</td>
            <td>{formatSize(res.file_size)}</td>
            <td>
              <button class="outline" type="button" on:click={() => copyShareLink(res.id)}>
                {copySuccess === res.id ? 'Link copied!' : 'Copy Link'}
              </button>
            </td>
            <td>
              <button class="outline secondary" type="button" on:click={() => handleDelete(res.id)}>
                Delete
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </figure>

  <!-- Recently Watched -->
  <details style="margin-top: 1.5rem;">
    <summary role="button" class="outline">Recently Watched ({watchHistory.length})</summary>
    {#if watchHistory.length > 0}
      <div style="margin-top: 0.5rem; display: flex; gap: 0.5rem;">
        <button class="outline secondary" style="font-size: 0.875rem;" on:click={handleClearHistory}>Clear All History</button>
      </div>
      <table role="grid">
        <thead>
          <tr>
            <th>Video</th>
            <th>Watched</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each watchHistory as record}
            <tr>
              <td>
                <a href="#/s/{record.id}/watch">{record.title || 'Untitled'}</a>
              </td>
              <td>{formatRelativeTime(record.watchedAt)}</td>
              <td>
                <button class="outline" on:click={() => clearWatchEntry(record.id)}>Clear</button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else}
      <p style="padding: 0.75rem 0;">No videos watched yet.</p>
    {/if}
  </details>
{/if}
