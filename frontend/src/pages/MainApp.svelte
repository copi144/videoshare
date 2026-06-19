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
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import MarkdownEditor from '../components/MarkdownEditor.svelte';

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

  let activeTab: 'browse' | 'history' | 'users' | 'categories' | 'playlists' = 'browse';

  // --- Category / Playlist filtering ---

  let selectedCategoryId: string = 'global';
  let selectedPlaylistId: string | null = null;
  let selectedResourceType: 'all' | 'video' | 'audio' | 'image' = 'all';
  let categories: Category[] = [];
  let playlists: Playlist[] = [];

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

  // --- File type filter (driven by top type selector) ---
  const videoAccept = 'video/mp4,video/webm,video/x-matroska,video/quicktime,video/x-msvideo,video/x-flv';
  const audioAccept = 'audio/mpeg,audio/mp4,audio/wav,audio/ogg,audio/flac,audio/aac';
  const imageAccept = 'image/jpeg,image/png,image/webp,image/gif';
  const allAccept = `${videoAccept},${audioAccept},${imageAccept}`;

  $: fileAccept = selectedResourceType === 'video' ? videoAccept
    : selectedResourceType === 'audio' ? audioAccept
    : selectedResourceType === 'image' ? imageAccept
    : allAccept;

  // --- Confirm modal ---

  let showConfirm = false;
  let confirmTitle = '';
  let confirmMessage = '';
  let confirmLabel = 'Confirm';
  let pendingConfirm: (() => Promise<void>) | null = null;

  function openConfirm(title: string, message: string, label: string, action: () => Promise<void>) {
    confirmTitle = title;
    confirmMessage = message;
    confirmLabel = label;
    pendingConfirm = action;
    showConfirm = true;
  }

  $: selectedCategory = categories.find((c) => c.id === uploadForm.category_id);
  $: isGlobal = selectedCategory ? selectedCategory.id === 'global' : false;

  // --- Data loading ---

  async function loadResources() {
    error = null;
    try {
      const params: { limit: number; offset: number; category_id?: string; playlist_id?: string; resource_type?: string } = { limit, offset };
      if (selectedPlaylistId) {
        params.playlist_id = selectedPlaylistId;
      } else if (selectedCategoryId) {
        params.category_id = selectedCategoryId;
      }
      if (selectedResourceType !== 'all') {
        params.resource_type = selectedResourceType;
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
      const data = await listPlaylists({ category_id: selectedCategoryId, playlist_type: selectedResourceType });
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

  function onTypeChange() {
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
    openConfirm(
      'Delete Video',
      'Are you sure you want to delete this video? This action cannot be undone.',
      'Delete',
      async () => {
        await deleteResource(id);
        await loadResources();
      }
    );
  }

  async function handleRetranscode(id: string) {
    openConfirm(
      'Retranscode Video',
      'Are you sure you want to retranscode this video?',
      'Retranscode',
      async () => {
        await retranscode(id);
        await loadResources();
      }
    );
  }

  async function handleBan(id: string) {
    openConfirm(
      'Ban Video',
      'Are you sure you want to ban this video? This will delete the video data and prevent re-upload of the same file.',
      'Ban',
      async () => {
        await banResource(id);
        await loadResources();
      }
    );
  }

  async function handleBatchDelete() {
    openConfirm(
      'Delete Selected Videos',
      `Delete ${selectedIds.size} selected videos? This action cannot be undone.`,
      'Delete All',
      async () => {
        await Promise.all(Array.from(selectedIds).map((id) => deleteResource(id)));
        selectedIds = new Set();
        selectMode = false;
        await loadResources();
      }
    );
  }

  async function handleBatchBan() {
    openConfirm(
      'Ban Selected Videos',
      `Ban ${selectedIds.size} selected videos? This will delete video data and prevent re-upload.`,
      'Ban All',
      async () => {
        await Promise.all(Array.from(selectedIds).map((id) => banResource(id)));
        selectedIds = new Set();
        selectMode = false;
        await loadResources();
      }
    );
  }

  async function handleBatchRetranscode() {
    openConfirm(
      'Retranscode Selected Videos',
      `Retranscode ${selectedIds.size} selected videos?`,
      'Retranscode All',
      async () => {
        await Promise.all(Array.from(selectedIds).map((id) => retranscode(id)));
        selectedIds = new Set();
        selectMode = false;
        await loadResources();
      }
    );
  }

  async function handleConfirmAction() {
    if (pendingConfirm) {
      try {
        await pendingConfirm();
      } catch (e: unknown) {
        error = e instanceof Error ? e.message : 'Action failed.';
      }
      pendingConfirm = null;
    }
    showConfirm = false;
  }

  function handleCancelConfirm() {
    showConfirm = false;
    pendingConfirm = null;
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
      <button
        class="tab"
        class:active={activeTab === 'categories'}
        on:click={() => { activeTab = 'categories'; }}
      >
        Categories
      </button>
      <button
        class="tab"
        class:active={activeTab === 'playlists'}
        on:click={() => { activeTab = 'playlists'; }}
      >
        Playlists
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
  </nav>

  <!-- Main content -->
  <div class="content-row">
    <!-- Content area -->
    <div class="content-area">
      {#if error}
        <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{error}</div>
      {/if}

      {#if activeTab === 'browse'}
        <!-- Action bar -->
        <div class="action-bar">
          <div class="action-bar-left">
            <select bind:value={selectedCategoryId} on:change={onCategoryChange}>
              {#each categories as cat}
                <option value={cat.id}>
                  {cat.name}{cat.id === 'global' ? ' (public)' : ''}
                </option>
              {/each}
            </select>
            <select bind:value={selectedResourceType} on:change={onTypeChange}>
              <option value="all">All</option>
              <option value="video">Video</option>
              <option value="audio">Audio</option>
              <option value="image">Image</option>
            </select>
          </div>
          <div class="action-bar-right">
            {#if selectMode && selectedIds.size > 0}
              <button class="action-btn retranscode-all" on:click={handleBatchRetranscode}>
                Re-transcode All
              </button>
              {#if userRole === 'admin'}
                <button class="action-btn ban-all" on:click={handleBatchBan}>
                  Ban All
                </button>
              {/if}
              <button class="action-btn delete-all" on:click={handleBatchDelete}>
                Delete All
              </button>
            {/if}
            <button class="action-btn select-btn" on:click={() => { selectMode = !selectMode; selectedIds = new Set(); }}>
              {selectMode ? 'Done' : 'Select'}
            </button>
          </div>
        </div>

        <!-- Resource section -->
        <div class="resource-section">

          {#if loading}
            <p aria-busy="true">Loading videos…</p>
          {:else if resources.length === 0}
            <p>
              {#if selectedPlaylistId}
                No videos in this playlist.
              {:else}
                Upload a video using the form in the Browse tab.
              {/if}
            </p>
          {:else}
            <table class="w-full text-left divide-y divide-gray-200">
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
                    <th class="share-col">Share Link</th>
                    <th class="actions-col">Actions</th>
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
                          <span class="text-red-600 font-bold">Banned</span>
                        {:else if res.transcode_status === 'done'}
                          <span class="text-indigo-600">Ready</span>
                        {:else if res.transcode_status === 'processing'}
                          <span class="text-yellow-600">Processing</span>
                        {:else if res.transcode_status === 'pending'}
                          <span class="text-yellow-600">Pending</span>
                        {:else if res.transcode_status === 'failed'}
                          <span class="text-red-600">Failed</span>
                        {:else}
                          <span class="text-gray-400">&mdash;</span>
                        {/if}
                      </td>
                      <td>{res.views}</td>
                      <td>{formatSize(res.file_size)}</td>
                      <td class="share-col">
                        <button
                          class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50"
                          type="button"
                          on:click={() => { copyShareLink(res.id); }}
                        >
                          {copySuccess === res.id ? 'Link copied!' : 'Copy Link'}
                        </button>
                      </td>
                      <td class="actions-col">
                        <button
                          class="row-action-btn row-action-retranscode"
                          type="button"
                          on:click={() => { handleRetranscode(res.id); }}
                        >
                          Re-transcode
                        </button>
                        {#if !res.banned}
                          <button
                            class="row-action-btn row-action-ban"
                            type="button"
                            on:click={() => { handleBan(res.id); }}
                          >
                            Ban
                          </button>
                        {/if}
                        <button
                          class="row-action-btn row-action-delete"
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

            <!-- Pagination -->
            <div class="pagination-bar">
              <span>{offset + 1}&ndash;{offset + resources.length} of {total}</span>
              <button
                class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                type="button"
                on:click={handlePrevPage}
                disabled={offset === 0}
              >
                Prev
              </button>
              <button
                class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                type="button"
                on:click={handleNextPage}
                disabled={offset + resources.length >= total}
              >
                Next
              </button>
            </div>
          {/if}
        </div>

        <!-- Upload form -->
        <div class="mt-6">
          <form on:submit|preventDefault={handleUpload} class="upload-form">
            <div class="upload-row-title">
              <span class="upload-label">upload</span>
              <input type="text" id="title" name="title" bind:value={uploadForm.title} placeholder="title" required />
            </div>
            <div>
              <label for="readme" class="block text-sm font-medium text-gray-700 mb-1">Readme (Markdown)</label>
              <MarkdownEditor
                bind:value={uploadForm.readme}
                placeholder="Optional markdown description..."
              />
            </div>
            <div class="upload-row-actions">
              <select id="category" name="category_id" bind:value={uploadForm.category_id} class="upload-category-select" required>
                <option value="">&mdash; Category &mdash;</option>
                {#each categories as cat}
                  <option value={cat.id}>
                    {cat.name}{cat.id === 'global' ? ' (public)' : ''}
                  </option>
                {/each}
              </select>
              <label class="file-input-label" for="file">
                {selectedFile ? selectedFile.name : 'Browse…'}
              </label>
              <input
                type="file"
                id="file"
                name="file"
                accept={fileAccept}
                on:change={onFileChange}
                class="file-input-hidden"
                required
              />
              <label class="upload-checkbox-label">
                <input type="checkbox" bind:checked={uploadForm.noTranscode} />
                Skip transcoding
              </label>
            </div>
            {#if !isGlobal && uploadForm.category_id}
              <div class="upload-row-actions">
                <input type="text" id="password" name="password" bind:value={uploadForm.password} placeholder="Password (required for this category)" class="upload-password-input" />
              </div>
            {/if}
            {#if uploadError}
              <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{uploadError}</div>
            {/if}
            <button type="submit" disabled={uploading || !selectedFile || !uploadForm.title.trim() || !uploadForm.category_id} class="upload-submit-btn">
              {uploading ? 'Uploading…' : 'upload'}
            </button>
          </form>
        </div>

      {:else if activeTab === 'history'}
        <TabHistory />

      {:else if activeTab === 'users' && userRole === 'admin'}
        <div class="users-section">
          <Users onError={setError} />
        </div>

      {:else if activeTab === 'categories'}
        <Categories onError={setError} />

      {:else if activeTab === 'playlists'}
        <Playlists onError={setError} />
      {/if}

      <ConfirmModal
        show={showConfirm}
        title={confirmTitle}
        message={confirmMessage}
        confirmLabel={confirmLabel}
        onConfirm={handleConfirmAction}
        onCancel={handleCancelConfirm}
      />
    </div>

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
    padding: 0.25rem 0;
    margin-bottom: 0;
    border-bottom: 1px solid #e5e7eb;
  }

  .tab-group {
    display: flex;
    gap: 0.25rem;
  }

  button.tab {
    background: none;
    border: none;
    padding: 0.25rem 0.6rem;
    font-size: 0.9rem;
    cursor: pointer;
    border-radius: 0.375rem;
    color: #6b7280;
  }

  .tab-group button {
    font-size: 0.9rem;
  }

  button.tab.active,
  .tab:hover {
    background: #6366f1;
    color: white;
  }

  button.tab.active {
    border-bottom: 3px solid #6366f1;
    border-radius: 0.375rem 0.375rem 0 0;
  }

  .content-row {
    display: flex;
    gap: 0.5rem;
    flex: 1;
  }

  .content-area {
    flex: 1;
    min-width: 0;
  }

  .action-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-top: 0.5rem;
    margin-bottom: 0.5rem;
    gap: 0.5rem;
  }

  .action-bar-left {
    flex: 0 0 auto;
    display: flex;
    gap: 0.5rem;
  }

  .action-bar-left select:first-child {
    min-width: 180px;
  }

  .action-bar-right {
    display: flex;
    gap: 0.25rem;
    align-items: center;
  }

  .action-btn {
    padding: 0.25rem 0.6rem;
    font-size: 0.8rem;
    border: 1px solid #d1d5db;
    border-radius: 0.25rem;
    background: white;
    color: #374151;
    cursor: pointer;
    white-space: nowrap;
  }

  .action-btn:hover {
    background: #f3f4f6;
  }

  .action-btn.retranscode-all {
    color: #6366f1;
    border-color: #c7d2fe;
  }

  .action-btn.retranscode-all:hover {
    background: #eef2ff;
  }

  .action-btn.ban-all {
    color: #dc2626;
    border-color: #fecaca;
  }

  .action-btn.ban-all:hover {
    background: #fef2f2;
  }

  .action-btn.delete-all {
    color: #dc2626;
    border-color: #fecaca;
  }

  .action-btn.delete-all:hover {
    background: #fef2f2;
  }

  .action-btn.select-btn {
    font-weight: 500;
  }

  .pagination-bar {
    margin-top: 0.25rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  figure {
    margin: 0;
  }

  @media (max-width: 768px) {
    .content-row {
      flex-direction: column;
    }
  }

  .row-action-btn {
    padding: 0.2rem 0.5rem;
    font-size: 0.75rem;
    border: 1px solid;
    border-radius: 0.25rem;
    background: white;
    cursor: pointer;
    white-space: nowrap;
  }

  .row-action-retranscode {
    color: #6366f1;
    border-color: #c7d2fe;
  }

  .row-action-retranscode:hover {
    background: #eef2ff;
  }

  .row-action-ban {
    color: #dc2626;
    border-color: #fecaca;
  }

  .row-action-ban:hover {
    background: #fef2f2;
  }

  .row-action-delete {
    color: #dc2626;
    border-color: #fecaca;
  }

  .row-action-delete:hover {
    background: #fef2f2;
  }

  @media (max-width: 900px) {
    .actions-col {
      display: none;
    }
  }

  @media (max-width: 600px) {
    .share-col {
      display: none;
    }
  }

  .upload-form {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .upload-row-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .upload-label {
    font-size: 0.85rem;
    font-weight: 500;
    color: #6b7280;
    white-space: nowrap;
  }

  .upload-row-title input {
    flex: 1;
  }

  .upload-row-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .upload-category-select {
    max-width: 180px;
  }

  .upload-checkbox-label {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.8rem;
    color: #6b7280;
    white-space: nowrap;
  }

  .upload-password-input {
    max-width: 300px;
  }

  .upload-submit-btn {
    padding: 0.35rem 1rem;
    font-size: 0.85rem;
    font-weight: 500;
    border: 1px solid #6366f1;
    border-radius: 0.25rem;
    background: #6366f1;
    color: white;
    cursor: pointer;
    white-space: nowrap;
  }

  .upload-submit-btn:hover {
    background: #4f46e5;
  }

  .upload-submit-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .file-input-label {
    display: inline-flex;
    align-items: center;
    padding: 0.35rem 0.75rem;
    font-size: 0.8rem;
    border: 1px solid #d1d5db;
    border-radius: 0.25rem;
    background: white;
    color: #374151;
    cursor: pointer;
    white-space: nowrap;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-input-label:hover {
    background: #f3f4f6;
  }

  .file-input-hidden {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border-width: 0;
  }
</style>
