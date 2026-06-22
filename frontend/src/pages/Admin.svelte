<script lang="ts">
  import { onMount } from 'svelte';
  import { listResources, uploadVideo, uploadFileWithProgress, deleteResource, retranscode, banResource, listCategories } from '../lib/api';
  import MarkdownEditor from '../components/MarkdownEditor.svelte';
  import Categories from './Categories.svelte';
  import Playlists from './Playlists.svelte';
  import Users from './Users.svelte';

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
    categories?: string[];
    transcode_status?: string;
    banned?: boolean;
  }

  interface Category {
    name: string;
    display_name: string;
    description?: string;
  }

  let resources: Resource[] = [];
  let categories: Category[] = [];
  let uploadForm = { title: '', readme: '', category_id: '', noTranscode: false };
  let selectedFile: File | null = null;
  let error: string | null = null;
  let uploadError: string | null = null;
  let loading = true;
  let uploading = false;
  let uploadProgress = 0;
  let copySuccess: string | null = null;
  let dragOver = false;

  // Pagination (local state only)
  let limit = 50;
  let offset = 0;
  let total = 0;

  function onFileChange(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0] ?? null;
    selectedFile = file;
    if (file && !uploadForm.title.trim()) {
      const name = file.name.replace(/\.[^.]+$/, '');
      uploadForm.title = name;
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    const file = e.dataTransfer?.files?.[0] ?? null;
    selectedFile = file;
    if (file && !uploadForm.title.trim()) {
      const name = file.name.replace(/\.[^.]+$/, '');
      uploadForm.title = name;
    }
  }

  $: selectedCategory = categories.find(c => c.name === uploadForm.category_id);
  $: isGlobal = selectedCategory ? selectedCategory.name === 'global' : false;

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  async function loadData() {
    error = null;
    try {
      const [resData, catData] = await Promise.all([
        listResources({ limit, offset }),
        listCategories(),
      ]);
      resources = resData.resources;
      total = resData.total;
      categories = catData.categories;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load data.';
    } finally {
      loading = false;
    }
  }

  function setError(msg: string) {
    error = msg;
  }

  onMount(async () => {
    await loadData();
  });

  async function handleUpload() {
    uploadError = null;
    uploadProgress = 0;
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

    uploading = true;
    try {
      const fd = new FormData();
      fd.append('file', selectedFile);
      fd.append('title', uploadForm.title.trim());
      fd.append('readme', uploadForm.readme.trim());
      fd.append('category_id', uploadForm.category_id);
      fd.append('no_transcode', uploadForm.noTranscode ? '1' : '0');
      await uploadFileWithProgress(fd, (pct) => { uploadProgress = pct; });
      // Reset form
      uploadForm = { title: '', readme: '', category_id: '', noTranscode: false };
      selectedFile = null;
      uploadProgress = 0;
      // Reload resources
      await loadData();
    } catch (e: unknown) {
      uploadError = e instanceof Error ? e.message : 'Upload failed.';
      uploadProgress = 0;
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
      await loadData();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to delete resource.';
    }
  }

  async function handleRetranscode(id: string) {
    try {
      await retranscode(id);
      await loadData();
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
      await loadData();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to ban resource.';
    }
  }

  function copyShareLink(id: string) {
    const url = `${window.location.origin}/#/v/${id}`;
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
  <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700 mb-4">{error}</div>
{/if}

<!-- Upload Form -->
<div class="rounded-lg border border-gray-200 bg-white p-4 mb-4">
  <h2 class="text-base font-semibold text-gray-900 mb-3">Upload Video</h2>
  <div
    class="space-y-3"
    class:border-indigo-300={dragOver}
    role="application"
    on:dragover|preventDefault={() => dragOver = true}
    on:dragleave={() => dragOver = false}
    on:drop|preventDefault={handleDrop}
  >
  <form on:submit|preventDefault={handleUpload}>
    <div>
      <label for="title" class="block text-sm font-medium text-gray-700 mb-1">Title</label>
      <input type="text" id="title" name="title" bind:value={uploadForm.title} required class="w-full" />
    </div>
    <div>
      <label for="readme" class="block text-sm font-medium text-gray-700 mb-1">Readme (Markdown)</label>
      <MarkdownEditor bind:value={uploadForm.readme} placeholder="Optional markdown description..." />
    </div>
    <div>
      <label for="category" class="block text-sm font-medium text-gray-700 mb-1">Category</label>
      <select id="category" name="category_id" bind:value={uploadForm.category_id} required class="w-full">
        <option value="">— Select a category —</option>
        {#each categories as cat}
          <option value={cat.name}>
            {cat.display_name || cat.name}{cat.name === 'global' ? ' (public)' : ''}
          </option>
        {/each}
      </select>
    </div>
    <div>
      <label for="file" class="block text-sm font-medium text-gray-700 mb-1">Video File</label>
      <input
        type="file"
        id="file"
        name="file"
        accept="video/mp4,video/webm,video/x-matroska,video/quicktime,video/x-msvideo,video/x-flv,audio/mpeg,audio/mp4,audio/wav,audio/ogg,audio/flac,audio/aac,image/jpeg,image/png,image/webp,image/gif"
        on:change={onFileChange}
        required
        class="w-full"
      />
    </div>
    <label class="inline-flex items-center gap-2 text-sm text-gray-700">
      <input type="checkbox" bind:checked={uploadForm.noTranscode} />
      Skip transcoding (serve original file directly)
    </label>
    {#if uploadError}
      <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{uploadError}</div>
    {/if}
    {#if uploading}
      <div class="w-full bg-gray-200 rounded-full h-2.5 mb-2">
        <div class="bg-indigo-600 h-2.5 rounded-full transition-all duration-300" style="width: {uploadProgress}%"></div>
      </div>
      <p class="text-xs text-gray-500 text-right mb-2">{uploadProgress}%</p>
    {/if}
    <div class="flex justify-end">
      <button type="submit" disabled={uploading} class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50" aria-busy={uploading}>
        {uploading ? 'Upload ' + uploadProgress + '%…' : 'Upload'}
      </button>
    </div>
  </form>
  </div>
</div>

<!-- Resources Table -->
<h2 class="text-base font-semibold text-gray-900 mb-3">My Videos</h2>
{#if loading}
  <p class="text-gray-500 text-sm">Loading videos…</p>
{:else if resources.length === 0}
  <p class="text-gray-500 text-sm">No videos yet. Upload one above.</p>
{:else}
  <div class="rounded-lg border border-gray-200 bg-white p-4 mb-4">
    <table class="w-full text-left text-sm">
      <thead>
        <tr class="border-b border-gray-200">
          <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Title</th>
          <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Category</th>
          <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Status</th>
          <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Views</th>
          <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Size</th>
          <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Share Link</th>
          <th class="py-2 text-xs font-medium text-gray-500 uppercase">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each resources as res}
          <tr class="border-b border-gray-100">
            <td class="py-2 pr-4">{res.title}</td>
            <td class="py-2 pr-4">
              {#if res.categories && res.categories.length > 0}
                <div class="flex flex-wrap gap-1">
                  {#each res.categories as cat}
                    <span class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600">{cat}</span>
                  {/each}
                </div>
              {:else}
                <span class="text-gray-500">&mdash;</span>
              {/if}
            </td>
            <td class="py-2 pr-4">
              {#if res.banned}
                <span class="inline-flex items-center rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">Banned</span>
              {:else if res.transcode_status === 'done'}
                <span class="inline-flex items-center rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">Ready</span>
              {:else if res.transcode_status === 'processing'}
                <span class="inline-flex items-center rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-700" aria-busy="true">Processing</span>
              {:else if res.transcode_status === 'pending'}
                <span class="inline-flex items-center rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-700">Pending</span>
              {:else if res.transcode_status === 'failed'}
                <span class="inline-flex items-center rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">Failed</span>
              {:else}
                <span class="text-gray-400">&mdash;</span>
              {/if}
            </td>
            <td class="py-2 pr-4">{res.views}</td>
            <td class="py-2 pr-4">{formatSize(res.file_size)}</td>
            <td class="py-2 pr-4">
              <button class="row-action-btn" type="button" on:click={() => copyShareLink(res.id)}>
                {copySuccess === res.id ? 'Link copied!' : 'Copy Link'}
              </button>
            </td>
            <td class="py-2">
              <div class="flex gap-1">
                <button class="row-action-btn" type="button" on:click={() => handleRetranscode(res.id)}>
                  Re-transcode
                </button>
                {#if !res.banned}
                  <button class="row-action-btn" type="button" on:click={() => handleBan(res.id)}>
                    Ban
                  </button>
                {/if}
                <button class="row-action-btn row-action-delete" type="button" on:click={() => handleDelete(res.id)}>
                  Delete
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
    <div class="mt-3 flex items-center gap-2 text-sm">
      <span class="text-gray-500">{offset + 1}–{offset + resources.length} of {total}</span>
      <button type="button" class="row-action-btn" on:click={() => { if (offset > 0) { offset = Math.max(0, offset - limit); loadData(); } }} disabled={offset === 0}>Prev</button>
      <button type="button" class="row-action-btn" on:click={() => { offset += limit; loadData(); }} disabled={offset + resources.length >= total}>Next</button>
    </div>
  </div>
{/if}

<!-- Management Sections -->
<details class="admin-details">
  <summary class="admin-summary">Category Management</summary>
  <Categories onError={setError} />
</details>

<details class="admin-details">
  <summary class="admin-summary">Playlist Management</summary>
  <Playlists onError={setError} />
</details>

<details class="admin-details">
  <summary class="admin-summary">User Management</summary>
  <Users onError={setError} />
</details>

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
  .row-action-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .row-action-delete {
    color: #dc2626;
    border-color: #fecaca;
  }
  .row-action-delete:hover:not(:disabled) {
    background: #fef2f2;
  }
  .admin-details {
    margin-top: 1rem;
    border: 1px solid #d1d5db;
    border-radius: 0.5rem;
    background: white;
    overflow: hidden;
  }
  .admin-summary {
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: #374151;
    cursor: pointer;
    background: #f9fafb;
    border-bottom: 1px solid #e5e7eb;
    user-select: none;
  }
  .admin-summary:hover {
    background: #f3f4f6;
  }
  .admin-details[open] .admin-summary {
    border-bottom: 1px solid #d1d5db;
  }
  .admin-details > :not(summary) {
    padding: 1rem;
  }
</style>
