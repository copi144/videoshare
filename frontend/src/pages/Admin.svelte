<script lang="ts">
  import { onMount } from 'svelte';
  import { listResources, uploadVideo, deleteResource, retranscode, banResource, listCategories } from '../lib/api';
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

  let resources: Resource[] = [];
  let categories: Category[] = [];
  let uploadForm = { title: '', readme: '', category_id: '', password: '' };
  let selectedFile: File | null = null;
  let error: string | null = null;
  let uploadError: string | null = null;
  let loading = true;
  let uploading = false;
  let copySuccess: string | null = null;

  // Pagination (local state only)
  let limit = 50;
  let offset = 0;
  let total = 0;

  function onFileChange(e: Event) {
    selectedFile = (e.target as HTMLInputElement).files?.[0] ?? null;
  }

  $: selectedCategory = categories.find(c => c.id === uploadForm.category_id);
  $: isGlobal = selectedCategory ? selectedCategory.id === 'global' : false;

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
      if (uploadForm.password.trim()) {
        fd.append('password', uploadForm.password.trim());
      }
      await uploadVideo(fd);
      // Reset form
      uploadForm = { title: '', readme: '', category_id: '', password: '' };
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
    const url = `${window.location.origin}/s/${id}`;
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

<!-- Upload Form -->
<article>
  <h2>Upload Video</h2>
  <form on:submit|preventDefault={handleUpload}>
    <label for="title">
      Title
      <input type="text" id="title" name="title" bind:value={uploadForm.title} required />
    </label>
    <label for="readme">
      Readme (Markdown)
      <textarea id="readme" name="readme" bind:value={uploadForm.readme} placeholder="Optional markdown description..."></textarea>
    </label>
    <label for="category">
      Category
      <select id="category" name="category_id" bind:value={uploadForm.category_id} required>
        <option value="">— Select a category —</option>
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
{:else if resources.length === 0}
  <p>No videos yet. Upload one above.</p>
{:else}
  <figure>
    <table role="grid">
      <thead>
        <tr>
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
              <button class="outline" type="button" on:click={() => copyShareLink(res.id)}>
                {copySuccess === res.id ? 'Link copied!' : 'Copy Link'}
              </button>
            </td>
            <td>
              <button class="outline" type="button" on:click={() => handleRetranscode(res.id)}>
                Re-transcode
              </button>
              {#if !res.banned}
                <button class="outline secondary" type="button" on:click={() => handleBan(res.id)}>
                  Ban
                </button>
              {/if}
              <button class="outline secondary" type="button" on:click={() => handleDelete(res.id)}>
                Delete
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </figure>

  <!-- Pagination -->
  <div style="margin-top: 0.5rem; display: flex; align-items: center; gap: 0.5rem;">
    <span>{offset + 1}–{offset + resources.length} of {total}</span>
    <button type="button" on:click={() => { if (offset > 0) { offset = Math.max(0, offset - limit); loadData(); } }} disabled={offset === 0}>Prev</button>
    <button type="button" on:click={() => { offset += limit; loadData(); }} disabled={offset + resources.length >= total}>Next</button>
  </div>
{/if}

<!-- Management Sections -->
<details style="margin-top: 1.5rem;">
  <summary role="button" class="outline secondary">Category Management</summary>
  <Categories onError={setError} />
</details>

<details style="margin-top: 1rem;">
  <summary role="button" class="outline secondary">Playlist Management</summary>
  <Playlists onError={setError} />
</details>

<details style="margin-top: 1rem;">
  <summary role="button" class="outline secondary">User Management</summary>
  <Users onError={setError} />
</details>
