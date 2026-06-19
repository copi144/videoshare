<script lang="ts">
  import { onMount } from 'svelte';
  import { user } from '../stores/auth';
  import {
    listResources,
    uploadVideo,
    deleteResource,
    retranscode,
    banResource,
    listCategories,
    listPlaylists,
  } from '../lib/api';
  import TabHistory from './TabHistory.svelte';
  import Categories from './Categories.svelte';
  import Playlists from './Playlists.svelte';
  import Users from './Users.svelte';

  // --- Types ---

  interface Resource {
    id: string;
    title: string;
    content_type: string;
    file_size: number;
    views: number;
    created_at: string;
    updated_at?: string;
    uploaded_by: string;
    uploaded_username: string;
    filename?: string;
    category_id: string;
    category_name: string;
    transcode_status?: string;
    banned?: boolean;
  }

  interface Category {
    id: string;
    name: string;
    description?: string;
  }

  interface Playlist {
    id: string;
    name: string;
    description?: string;
    category_id: string;
  }

  // --- Auth ---

  $: userRole = $user?.role || '';

  // --- Tab state ---

  let activeTab: 'browse' | 'history' | 'users' = 'browse';

  // --- Category / Playlist filtering ---

  let selectedCategoryId: string = 'global';
  let selectedPlaylistId: string | null = null;
  let categories: Category[] = [];
  let playlists: Playlist[] = [];

  $: categoryPlaylists = playlists.filter((pl) => pl.category_id === selectedCategoryId);

  // --- Sidebar ---

  let showSidebar = false;

  // --- Resources ---

  let resources: Resource[] = [];
  let error: string | null = null;
  let loading = true;
  let limit = 50;
  let offset = 0;
  let total = 0;

  // --- Multi-select ---

  let selectMode = false;
  let selectedIds: Set<string> = new Set();

  // --- Upload form ---

  let uploadForm = { title: '', readme: '', category_id: '', password: '', noTranscode: false };
  let selectedFile: File | null = null;
  let uploadError: string | null = null;
  let uploading = false;
  let copySuccess: string | null = null;

  $: selectedCategory = categories.find((c) => c.id === uploadForm.category_id);
  $: isGlobal = selectedCategory ? selectedCategory.id === 'global' : false;

  // --- Data loading ---

  async function loadResources() {
    error = null;
    try {
      const params: { limit: number; offset: number; category_id?: string; playlist_id?: string } = { limit, offset };
      if (selectedPlaylistId) {
        params.playlist_id = selectedPlaylistId;
      } else if (selectedCategoryId) {
        params.category_id = selectedCategoryId;
      }
      const data = await listResources(params);
      resources = data.resources;
      total = data.total;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load resources.';
    }
  }

  async function loadCategories() {
    try {
      const data = await listCategories();
      categories = data.categories;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load categories.';
    }
  }

  async function loadPlaylists() {
    try {
      const data = await listPlaylists({ category_id: selectedCategoryId });
      playlists = data.playlists;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load playlists.';
    }
  }

  async function loadAll() {
    loading = true;
    error = null;
    try {
      await Promise.all([loadResources(), loadCategories(), loadPlaylists()]);
    } catch {
      // Errors are handled per-function
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadAll();
  });

  // --- Event handlers ---

  function onCategoryChange() {
    selectedPlaylistId = null;
    offset = 0;
    loadResources();
    loadPlaylists();
  }

  function onFileChange(e: Event) {
    selectedFile = (e.target as HTMLInputElement).files?.[0] ?? null;
  }

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
      fd.append('readme', uploadForm.readme.trim());
      fd.append('category_id', uploadForm.category_id);
      fd.append('no_transcode', uploadForm.noTranscode ? '1' : '0');
      if (uploadForm.password.trim()) {
        fd.append('password', uploadForm.password.trim());
      }
      await uploadVideo(fd);
      uploadForm = { title: '', readme: '', category_id: '', password: '', noTranscode: false };
      selectedFile = null;
      await loadResources();
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
      await loadResources();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to delete resource.';
    }
  }

  async function handleRetranscode(id: string) {
    try {
      await retranscode(id);
      await loadResources();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to retranscode video.';
    }
  }

  async function handleBan(id: string) {
    if (!confirm('Are you sure you want to ban this video? This will delete the video data and prevent re-upload of the same file.')) {
      return;
    }
    try {
      await banResource(id);
      await loadResources();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to ban resource.';
    }
  }

  async function handleBatchDelete() {
    if (!confirm(`Delete ${selectedIds.size} selected videos? This action cannot be undone.`)) {
      return;
    }
    try {
      await Promise.all(Array.from(selectedIds).map((id) => deleteResource(id)));
      selectedIds = new Set();
      selectMode = false;
      await loadResources();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to delete selected resources.';
    }
  }

  async function handleBatchBan() {
    if (!confirm(`Ban ${selectedIds.size} selected videos? This will delete video data and prevent re-upload.`)) {
      return;
    }
    try {
      await Promise.all(Array.from(selectedIds).map((id) => banResource(id)));
      selectedIds = new Set();
      selectMode = false;
      await loadResources();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to ban selected resources.';
    }
  }

  function toggleSelect(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    selectedIds = next;
  }

  function copyShareLink(id: string) {
    const url = `${window.location.origin}/s/${id}`;
    navigator.clipboard.writeText(url).then(
      () => {
        copySuccess = id;
        setTimeout(() => {
          copySuccess = null;
        }, 2000);
      },
      () => {
        // Fallback for older browsers
        const ta = document.createElement('textarea');
        ta.value = url;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
        copySuccess = id;
        setTimeout(() => {
          copySuccess = null;
        }, 2000);
      },
    );
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  function setError(msg: string) {
    error = msg;
  }

  function handlePrevPage() {
    if (offset > 0) {
      offset = Math.max(0, offset - limit);
      loadResources();
    }
  }

  function handleNextPage() {
    offset += limit;
    loadResources();
  }
</script>

<div class="app-layout">
  <!-- Top tab bar -->
  <nav class="tab-bar">
    <div class="tab-group">
      <button
        class="tab"
        class:active={activeTab === 'browse'}
        on:click={() => { activeTab = 'browse'; }}
      >
        Browse
      </button>
      <button
        class="tab"
        class:active={activeTab === 'history'}
        on:click={() => { activeTab = 'history'; }}
      >
        History
      </button>
      {#if userRole === 'admin'}
        <button
          class="tab"
          class:active={activeTab === 'users'}
          on:click={() => { activeTab = 'users'; }}
        >
          Users
        </button>
      {/if}
    </div>
    <div class="tab-actions">
      <button class="outline" on:click={() => { showSidebar = !showSidebar; }}>
        {showSidebar ? 'Close Manage' : 'Manage'}
      </button>
    </div>
  </nav>

  <!-- Main content and sidebar -->
  <div class="content-row">
    <!-- Content area -->
    <div class="content-area" class:with-sidebar={showSidebar}>
      {#if error}
        <article class="error-box">{error}</article>
      {/if}

      {#if activeTab === 'browse'}
        <!-- Category + Playlist selectors -->
        <div class="selector-bar">
          <select bind:value={selectedCategoryId} on:change={onCategoryChange}>
            {#each categories as cat}
              <option value={cat.id}>
                {cat.name}{cat.id === 'global' ? ' (public)' : ''}
              </option>
            {/each}
          </select>
          {#if categoryPlaylists.length > 0}
            <select bind:value={selectedPlaylistId} on:change={loadResources}>
              <option value={null}>All videos in category</option>
              {#each categoryPlaylists as pl}
                <option value={pl.id}>{pl.name}</option>
              {/each}
            </select>
          {/if}
        </div>

        <!-- Resource section -->
        <div class="resource-section">
          <div class="section-header">
            <h2>Videos</h2>
            <button class="outline" on:click={() => { selectMode = !selectMode; selectedIds = new Set(); }}>
              {selectMode ? 'Done Selecting' : 'Select'}
            </button>
          </div>

          <!-- Batch action bar -->
          {#if selectMode && selectedIds.size > 0}
            <div class="batch-bar">
              <span>{selectedIds.size} selected</span>
              <button on:click={handleBatchDelete}>Delete Selected</button>
              {#if userRole === 'admin'}
                <button on:click={handleBatchBan}>Ban Selected</button>
              {/if}
            </div>
          {/if}

          {#if loading}
            <p aria-busy="true">Loading videos…</p>
          {:else if resources.length === 0}
            <p>
              {#if selectedPlaylistId}
                No videos in this playlist.
              {:else}
                No videos yet. Upload one from the Manage panel.
              {/if}
            </p>
          {:else}
            <figure>
              <table role="grid">
                <thead>
                  <tr>
                    {#if selectMode}
                      <th></th>
                    {/if}
                    <th>Title</th>
                    <th>Category</th>
                    <th>Status</th>
                    <th>Views</th>
                    <th>Size</th>
                    <th>Share Link</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {#each resources as res}
                    <tr>
                      {#if selectMode}
                        <td>
                          <input
                            type="checkbox"
                            checked={selectedIds.has(res.id)}
                            on:change={() => { toggleSelect(res.id); }}
                          />
                        </td>
                      {/if}
                      <td>{res.title}</td>
                      <td>{res.category_name}</td>
                      <td>
                        {#if res.banned}
                          <span style="color: red; font-weight: bold;">Banned</span>
                        {:else if res.transcode_status === 'done'}
                          <span style="color: var(--primary);">Ready</span>
                        {:else if res.transcode_status === 'processing'}
                          <span style="color: var(--warning);" aria-busy="true">Processing</span>
                        {:else if res.transcode_status === 'pending'}
                          <span style="color: var(--warning);">Pending</span>
                        {:else if res.transcode_status === 'failed'}
                          <span style="color: var(--invalid);">Failed</span>
                        {:else}
                          <span style="color: var(--muted-color, #888);">&mdash;</span>
                        {/if}
                      </td>
                      <td>{res.views}</td>
                      <td>{formatSize(res.file_size)}</td>
                      <td>
                        <button
                          class="outline"
                          type="button"
                          on:click={() => { copyShareLink(res.id); }}
                        >
                          {copySuccess === res.id ? 'Link copied!' : 'Copy Link'}
                        </button>
                      </td>
                      <td>
                        <button
                          class="outline"
                          type="button"
                          on:click={() => { handleRetranscode(res.id); }}
                        >
                          Re-transcode
                        </button>
                        {#if !res.banned}
                          <button
                            class="outline secondary"
                            type="button"
                            on:click={() => { handleBan(res.id); }}
                          >
                            Ban
                          </button>
                        {/if}
                        <button
                          class="outline secondary"
                          type="button"
                          on:click={() => { handleDelete(res.id); }}
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </figure>

            <!-- Pagination -->
            <div class="pagination-bar">
              <span>{offset + 1}&ndash;{offset + resources.length} of {total}</span>
              <button
                type="button"
                on:click={handlePrevPage}
                disabled={offset === 0}
              >
                Prev
              </button>
              <button
                type="button"
                on:click={handleNextPage}
                disabled={offset + resources.length >= total}
              >
                Next
              </button>
            </div>
          {/if}
        </div>

      {:else if activeTab === 'history'}
        <TabHistory />

      {:else if activeTab === 'users' && userRole === 'admin'}
        <div class="users-section">
          <h2>User Management</h2>
          <Users onError={setError} />
        </div>
      {/if}
    </div>

    <!-- Right sidebar (collapsible) -->
    {#if showSidebar}
      <aside class="sidebar">
        <!-- Upload form -->
        <details open>
          <summary role="button" class="outline secondary">Upload Video</summary>
          <form on:submit|preventDefault={handleUpload}>
            <label for="title">
              Title
              <input type="text" id="title" name="title" bind:value={uploadForm.title} required />
            </label>
            <label for="readme">
              Readme (Markdown)
              <textarea
                id="readme"
                name="readme"
                bind:value={uploadForm.readme}
                placeholder="Optional markdown description..."
              ></textarea>
            </label>
            <label for="category">
              Category
              <select id="category" name="category_id" bind:value={uploadForm.category_id} required>
                <option value="">&mdash; Select a category &mdash;</option>
                {#each categories as cat}
                  <option value={cat.id}>
                    {cat.name}{cat.id === 'global' ? ' (public)' : ''}
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
            <label>
              <input type="checkbox" bind:checked={uploadForm.noTranscode} />
              Skip transcoding (serve original file directly)
            </label>
            {#if uploadError}
              <article class="error-box">{uploadError}</article>
            {/if}
            <button type="submit" disabled={uploading} aria-busy={uploading}>
              {uploading ? 'Uploading…' : 'Upload'}
            </button>
          </form>
        </details>

        <!-- Category management -->
        <details>
          <summary role="button" class="outline secondary">Categories</summary>
          <Categories onError={setError} />
        </details>

        <!-- Playlist management -->
        <details>
          <summary role="button" class="outline secondary">Playlists</summary>
          <Playlists onError={setError} />
        </details>

        {#if userRole === 'admin'}
          <!-- User management -->
          <details>
            <summary role="button" class="outline secondary">Users</summary>
            <Users onError={setError} />
          </details>
        {/if}
      </aside>
    {/if}
  </div>
</div>

<style>
  .app-layout {
    display: flex;
    flex-direction: column;
    min-height: 80vh;
  }

  .tab-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0;
    margin-bottom: 1rem;
    border-bottom: 1px solid var(--muted-border-color);
  }

  .tab-group {
    display: flex;
    gap: 0.25rem;
  }

  .tab {
    background: none;
    border: none;
    padding: 0.5rem 1rem;
    cursor: pointer;
    border-radius: var(--border-radius);
    color: var(--muted-color);
  }

  .tab.active,
  .tab:hover {
    background: var(--primary);
    color: var(--primary-inverse);
  }

  .content-row {
    display: flex;
    gap: 1.5rem;
    flex: 1;
  }

  .content-area {
    flex: 1;
    min-width: 0;
  }

  .content-area.with-sidebar {
    max-width: calc(100% - 380px);
  }

  .sidebar {
    width: 360px;
    flex-shrink: 0;
    border-left: 1px solid var(--muted-border-color);
    padding-left: 1rem;
    max-height: calc(100vh - 150px);
    overflow-y: auto;
  }

  .selector-bar {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .selector-bar select {
    flex: 1;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .batch-bar {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    padding: 0.5rem;
    background: var(--card-background-color);
    border-radius: var(--border-radius);
    margin-bottom: 0.5rem;
  }

  .pagination-bar {
    margin-top: 0.5rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .error-box {
    background: var(--form-element-invalid-border-color, #d32f2f);
    color: white;
    padding: 0.5rem 1rem;
    border-radius: var(--border-radius);
    margin-bottom: 1rem;
  }

  @media (max-width: 768px) {
    .content-row {
      flex-direction: column;
    }

    .content-area.with-sidebar {
      max-width: 100%;
    }

    .sidebar {
      width: 100%;
      border-left: none;
      padding-left: 0;
      max-height: none;
      overflow-y: visible;
    }

    .selector-bar {
      flex-direction: column;
    }
  }
</style>
