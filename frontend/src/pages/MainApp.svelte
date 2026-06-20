<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { user, isAuthenticated, apiToken, checkAuth, startHeartbeat, stopHeartbeat } from '../stores/auth';
  import { logout, setApiToken } from '../lib/api';
  import {
    listResources,
    getResource,
    uploadVideo,
    deleteResource,
    retranscode,
    banResource,
    listCategories,
    listPlaylists,
    createShareLink,
    listShareLinks,
    deleteShareLink,
    getShareLinkResources,
  } from '../lib/api';
  import TabHistory from './TabHistory.svelte';
  import Categories from './Categories.svelte';
  import Playlists from './Playlists.svelte';
  import Users from './Users.svelte';
  import ConfirmModal from '../components/ConfirmModal.svelte';
  import MarkdownEditor from '../components/MarkdownEditor.svelte';
  import Login from './Login.svelte';

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
    categories?: string[];
    transcode_status?: string;
    banned?: boolean;
  }

  interface Category {
    name: string;
    display_name: string;
    description?: string;
  }

  interface Playlist {
    name: string;
    description?: string;
    category_name: string;
  }

  // --- Auth ---

  $: isAdmin = $user?.is_admin || false;

  // --- Login modal ---

  let showLoginModal = false;

  async function handleLoginSuccess() {
    showLoginModal = false;
    await checkAuth();
    startHeartbeat();
    await loadAll();
  }

  async function handleLogout() {
    try {
      await logout();
    } catch {
      // Even if the API call fails, clear local state
    }
    setApiToken(null);
    apiToken.set(null);
    user.set(null);
    isAuthenticated.set(false);
    localStorage.removeItem('videoshare_api_token');
    stopHeartbeat();
    await loadAll();
  }

  // --- Initial navigation from hash routes ---

  export let initialCategory: string | null = null;
  export let initialPlaylist: string | null = null;

  // --- Shared mode ---

  export let sharedMode = false;
  export let shareTargetType: string | null = null;
  export let shareTargetId: string | null = null;
  export let shareTargetName: string | null = null;
  export let shareLinkId: string | null = null;
  export let shareLinkPassword: string | null = null;

  // --- Tab state ---

  let activeTab: 'browse' | 'history' | 'users' | 'categories' | 'playlists' = 'browse';

  // --- Category / Playlist filtering ---

  let selectedCategoryId: string = '';
  let selectedPlaylistComposite: string = ''; // format: "category:name" or "" for all

  function parsePlaylistComposite(composite: string): { category: string; name: string } {
    if (!composite) return { category: '', name: '' };
    const sep = composite.indexOf(':');
    if (sep === -1) return { category: '', name: composite };
    return { category: composite.substring(0, sep), name: composite.substring(sep + 1) };
  }
  let selectedResourceType: 'all' | 'video' | 'audio' | 'image' = 'all';
  let viewMode: 'list' | 'card' = 'list';
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

  let uploadForm = { title: '', readme: '', category_id: '', noTranscode: false };
  let selectedFile: File | null = null;
  let uploadError: string | null = null;
  let uploading = false;
  let copySuccess: string | null = null;
  let dragOver = false;
  
  // --- Create Link modal ---
  
  let showCreateLink = false;
  let createLinkResourceId = '';
  let createLinkExpiry = 1440;
  let createLinkExpiryMode: 'preset' | 'custom' = 'preset';
  let createLinkCustomMinutes = 1440;
  let createLinkUrl = '';
  let createLinkLoading = false;
  let createLinkError: string | null = null;

  $: if (createLinkExpiryMode === 'custom') {
    createLinkExpiry = createLinkCustomMinutes;
  }

  // --- Manage Links modal ---

  let manageLinkResourceId = '';
  let showManageLinks = false;
  let existingLinks: Array<{resource_id: string; password: string; expires_at: string | null; created_by: string; created_at: string}> = [];
  let manageLinksLoading = false;
  let manageLinksError: string | null = null;
  let deleteLinkId: string | null = null;

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

  $: selectedCategory = categories.find((c) => c.name === uploadForm.category_id);

  $: if (initialCategory) {
    selectedCategoryId = initialCategory;
    selectedPlaylistComposite = '';
    selectedResourceType = 'all';
    activeTab = 'browse';
    offset = 0;
    loadResources();
    initialCategory = null; // prevent re-trigger
  } else if (initialPlaylist) {
    selectedPlaylistComposite = initialPlaylist; // initialPlaylist is already "category:name"
    selectedResourceType = 'all';
    activeTab = 'browse';
    offset = 0;
    loadResources();
    initialPlaylist = null; // prevent re-trigger
  }

  // --- Data loading ---

  async function loadResources() {
    error = null;
    try {
      if (sharedMode && shareLinkId && shareLinkPassword) {
        const data = await getShareLinkResources(shareLinkId, shareLinkPassword);
        resources = data.resources || [];
        total = resources.length;
        loading = false;
        return;
      }
      const params: { limit: number; offset: number; category_name?: string; resource_type?: string } = { limit, offset };
      if (selectedPlaylistComposite) {
        const plInfo = parsePlaylistComposite(selectedPlaylistComposite);
        params.category_name = plInfo.category;
      } else if (selectedCategoryId) {
        params.category_name = selectedCategoryId;
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
      const data = await listPlaylists({ category_name: selectedCategoryId, playlist_type: selectedResourceType });
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

  onMount(async () => {
    if (sharedMode) {
      await loadResources();
      return;
    }
    await checkAuth();
    if ($isAuthenticated) {
      startHeartbeat();
    }
    await loadAll();
  });

  onDestroy(() => {
    stopHeartbeat();
  });

  // --- Event handlers ---

  function onCategoryChange() {
    selectedPlaylistComposite = '';
    offset = 0;
    loadResources();
    loadPlaylists();
  }

  function onTypeChange() {
    selectedPlaylistComposite = '';
    offset = 0;
    loadResources();
    loadPlaylists();
  }

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

    uploading = true;
    try {
      const fd = new FormData();
      fd.append('file', selectedFile);
      fd.append('title', uploadForm.title.trim());
      fd.append('readme', uploadForm.readme.trim());
      fd.append('category_id', uploadForm.category_id);
      fd.append('no_transcode', uploadForm.noTranscode ? '1' : '0');
      await uploadVideo(fd);
      uploadForm = { title: '', readme: '', category_id: '', noTranscode: false };
      selectedFile = null;
      await loadResources();
    } catch (e: unknown) {
      uploadError = e instanceof Error ? e.message : 'Upload failed.';
    } finally {
      uploading = false;
    }
  }

  async function handleDelete(id: string) {
    try {
      const detail = await getResource(id);
      const cats = (detail as any).categories;

      if (Array.isArray(cats) && cats.length > 1 && selectedCategoryId) {
        const otherCats = cats.filter((c: string) => c !== selectedCategoryId);
        openConfirm(
          'Unlink or Delete?',
          `This resource is in ${cats.length} categories: ${cats.join(', ')}. ` +
          `Remove from "${selectedCategoryId}" only? (File will NOT be deleted.)`,
          'Unlink',
          async () => {
            await deleteResource(id, selectedCategoryId);
            await loadResources();
          }
        );
      } else {
        openConfirm(
          'Delete Resource',
          'The file will be permanently removed from disk.',
          'Delete',
          async () => {
            await deleteResource(id);
            await loadResources();
          }
        );
      }
    } catch {
      openConfirm(
        'Delete Resource',
        'Are you sure? This will permanently delete the file.',
        'Delete',
        async () => {
          await deleteResource(id);
          await loadResources();
        }
      );
    }
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
      'Delete Selected',
      `Delete ${selectedIds.size} selected resources? Files will be permanently removed from disk.`,
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
    const url = `${window.location.origin}/#/v/${id}`;
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

  async function openCreateLink(id: string) {
    createLinkResourceId = id;
    createLinkExpiry = 1440;
    createLinkExpiryMode = 'preset';
    createLinkCustomMinutes = 1440;
    createLinkUrl = '';
    createLinkError = null;
    showCreateLink = true;
  }

  async function handleCreateLink() {
    createLinkLoading = true;
    createLinkError = null;
    try {
      const result = await createShareLink(createLinkResourceId, createLinkExpiry);
      if (result.url) {
        createLinkUrl = result.url;
      }
    } catch (e: unknown) {
      createLinkError = e instanceof Error ? e.message : 'Failed to create link.';
    } finally {
      createLinkLoading = false;
    }
  }

  function onExpiryModeChange() {
    if (createLinkExpiryMode === 'custom') {
      createLinkCustomMinutes = createLinkExpiry;
    }
  }

  function closeCreateLink() {
    showCreateLink = false;
    createLinkResourceId = '';
    createLinkUrl = '';
    createLinkError = null;
  }

  async function openManageLinks(id: string) {
    manageLinkResourceId = id;
    showManageLinks = true;
    manageLinksLoading = true;
    manageLinksError = null;
    try {
      const data = await listShareLinks(id);
      existingLinks = data.share_links;
    } catch (e: unknown) {
      manageLinksError = e instanceof Error ? e.message : 'Failed to load share links.';
      existingLinks = [];
    } finally {
      manageLinksLoading = false;
    }
  }

  function closeManageLinks() {
    showManageLinks = false;
    manageLinkResourceId = '';
    existingLinks = [];
    manageLinksError = null;
    deleteLinkId = null;
  }

  async function handleDeleteLink(password: string) {
    if (!manageLinkResourceId) return;
    deleteLinkId = password;
    try {
      await deleteShareLink(manageLinkResourceId, password);
      existingLinks = existingLinks.filter(l => l.password !== password);
    } catch (e: unknown) {
      manageLinksError = e instanceof Error ? e.message : 'Failed to delete link.';
    } finally {
      deleteLinkId = null;
    }
  }

  function formatDateTime(iso: string | null): string {
    if (!iso) return 'Never';
    return new Date(iso).toLocaleString();
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  function formatDuration(minutes: number): string {
    if (minutes < 1) return '0 minutes';
    if (minutes < 60) return `${minutes} minute${minutes !== 1 ? 's' : ''}`;
    if (minutes < 1440) {
      const hours = Math.floor(minutes / 60);
      const mins = minutes % 60;
      if (mins === 0) return `${hours} hour${hours !== 1 ? 's' : ''}`;
      return `${hours} hour${hours !== 1 ? 's' : ''}, ${mins} minute${mins !== 1 ? 's' : ''}`;
    }
    const days = Math.floor(minutes / 1440);
    const remaining = minutes % 1440;
    const hours = Math.floor(remaining / 60);
    const mins = remaining % 60;
    const parts: string[] = [];
    parts.push(`${days} day${days !== 1 ? 's' : ''}`);
    if (hours > 0) parts.push(`${hours} hour${hours !== 1 ? 's' : ''}`);
    if (mins > 0) parts.push(`${mins} minute${mins !== 1 ? 's' : ''}`);
    return parts.join(', ');
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
      {#if !sharedMode}
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
        {#if isAdmin}
          <button
            class="tab"
            class:active={activeTab === 'users'}
            on:click={() => { activeTab = 'users'; }}
          >
            Users
          </button>
        {/if}
      {/if}
    </div>
    <div class="user-section">
      {#if $isAuthenticated}
        <span class="username-label">{$user?.name}</span>
        <button class="login-btn" on:click={handleLogout}>Logout</button>
      {:else}
        <button class="login-btn" on:click={() => { showLoginModal = true; }}>Login</button>
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
        <div class="rounded-lg border border-gray-200 bg-white p-4 mb-4">
          <!-- Action bar -->
          <div class="action-bar">
          <div class="action-bar-left">
            {#if sharedMode}
              <span class="inline-flex items-center px-3 py-1 text-sm text-gray-700 bg-gray-50 rounded-md border border-gray-200">
                {shareTargetName}
              </span>
            {:else}
              <select bind:value={selectedCategoryId} on:change={onCategoryChange}>
                <option value="">All categories</option>
                {#each categories as cat}
                  <option value={cat.name}>
                    {cat.display_name || cat.name}{cat.name === 'global' ? ' (public)' : ''}
                  </option>
                {/each}
              </select>
            {/if}
            {#if !sharedMode}
              <select bind:value={selectedPlaylistComposite} on:change={() => { offset = 0; loadResources(); }}>
                <option value="">All playlists</option>
                {#each playlists as pl}
                  <option value="{pl.category_name}:{pl.name}">{pl.name} ({pl.category_name})</option>
                {/each}
              </select>
            {/if}
            <select bind:value={selectedResourceType} on:change={onTypeChange}>
              <option value="all">All file types</option>
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
      {#if isAdmin}
                <button class="action-btn ban-all" on:click={handleBatchBan}>
                  Ban All
                </button>
              {/if}
              <button class="action-btn delete-all" on:click={handleBatchDelete}>
                Delete All
              </button>
            {/if}
            <div class="inline-flex rounded-md shadow-sm">
              <button
                type="button"
                class="view-toggle-btn {viewMode === 'list' ? 'view-toggle-active' : ''}"
                on:click={() => viewMode = 'list'}
                title="List view"
              >
                List
              </button>
              <button
                type="button"
                class="view-toggle-btn {viewMode === 'card' ? 'view-toggle-active' : ''}"
                on:click={() => viewMode = 'card'}
                title="Card view"
              >
                Cards
              </button>
            </div>
            {#if $isAuthenticated}
            <button class="action-btn select-btn" on:click={() => { selectMode = !selectMode; selectedIds = new Set(); }}>
              {selectMode ? 'Done' : 'Select'}
            </button>
            {/if}
          </div>
        </div>

        <!-- Resource section -->
        <div class="resource-section">

          {#if loading}
            <p class="text-gray-500 text-sm" aria-busy="true">Loading videos…</p>
          {:else if resources.length === 0}
            <p class="text-gray-500 text-sm">
              {#if selectedPlaylistComposite}
                No videos in this playlist.
              {:else}
                Upload a video using the form in the Browse tab.
              {/if}
            </p>
          {:else}
            {#if viewMode === 'list'}
              <!-- Table view -->
              <table class="w-full text-left text-sm">
                <thead>
                  <tr class="border-b border-gray-200">
                    {#if selectMode}
                      <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase"></th>
                    {/if}
                    <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Title</th>
                    <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Download</th>
                    <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Status</th>
                    <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Views</th>
                    <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Size</th>
                    <th class="py-2 text-xs font-medium text-gray-500 uppercase">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {#each resources as res}
                    <tr class="border-b border-gray-100">
                      {#if selectMode}
                        <td class="py-2 pr-4">
                          <input
                            type="checkbox"
                            checked={selectedIds.has(res.id)}
                            on:change={() => { toggleSelect(res.id); }}
                          />
                        </td>
                      {/if}
                      <td class="py-2 pr-4"><a href="/#/v/{res.id}" class="text-indigo-600 hover:text-indigo-800 underline">{res.title}</a></td>
                      <td class="py-2 pr-4">
                        {#if res.resource_type === 'video'}
                          <a href="/v/{res.id}/download" target="_blank" class="row-action-btn">Download</a>
                        {:else if res.resource_type === 'audio'}
                          <a href="/a/{res.id}" target="_blank" class="row-action-btn">Download</a>
                        {:else if res.resource_type === 'image'}
                          <a href="/i/{res.id}" target="_blank" class="row-action-btn">Download</a>
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
                      <td class="py-2 pr-4 text-gray-500">{formatSize(res.file_size)}</td>
                      <td class="py-2">
                        <div class="flex gap-1">
                          {#if res.categories && res.categories.includes('global')}
                            <button class="row-action-btn" type="button" on:click={() => { copyShareLink(res.id); }}>
                              {copySuccess === res.id ? 'Link copied!' : 'Copy Link'}
                            </button>
                          {:else if $isAuthenticated}
                            <button class="row-action-btn" type="button" on:click={() => { openCreateLink(res.id); }}>
                              Create Link
                            </button>
                            <button class="row-action-btn" type="button" on:click={() => { openManageLinks(res.id); }}>
                              Manage
                            </button>
                          {/if}
                          {#if $isAuthenticated}
                            <button class="row-action-btn row-action-retranscode" type="button" on:click={() => { handleRetranscode(res.id); }}>
                              Re-transcode
                            </button>
                            {#if !res.banned}
                              <button class="row-action-btn row-action-ban" type="button" on:click={() => { handleBan(res.id); }}>
                                Ban
                              </button>
                            {/if}
                            <button class="row-action-btn row-action-delete" type="button" on:click={() => { handleDelete(res.id); }}>
                              Delete
                            </button>
                          {/if}
                        </div>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {:else}
              <!-- Card view -->
              <div class="resource-grid">
                {#each resources as res}
                  <div class="resource-card">
                    {#if selectMode}
                      <div class="card-checkbox">
                        <input
                          type="checkbox"
                          checked={selectedIds.has(res.id)}
                          on:change={() => { toggleSelect(res.id); }}
                        />
                      </div>
                    {/if}
                    <div class="card-title">
                      <a href="/#/v/{res.id}" class="text-indigo-600 hover:text-indigo-800 underline font-medium">{res.title}</a>
                    </div>
                    <div class="card-meta">
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
                    </div>
                    <div class="card-stats">
                      <span>{res.views} views</span>
                      <span>{formatSize(res.file_size)}</span>
                    </div>
                    <div class="card-actions">
                      {#if res.resource_type === 'video'}
                        <a href="/v/{res.id}/download" target="_blank" class="card-action-btn">Download</a>
                      {:else if res.resource_type === 'audio'}
                        <a href="/a/{res.id}" target="_blank" class="card-action-btn">Download</a>
                      {:else if res.resource_type === 'image'}
                        <a href="/i/{res.id}" target="_blank" class="card-action-btn">Download</a>
                      {/if}
                      {#if res.categories && res.categories.includes('global')}
                        <button class="card-action-btn" type="button" on:click={() => { copyShareLink(res.id); }}>
                          {copySuccess === res.id ? 'Link copied!' : 'Copy Link'}
                        </button>
                      {:else if $isAuthenticated}
                        <button class="card-action-btn" type="button" on:click={() => { openCreateLink(res.id); }}>
                          Create Link
                        </button>
                        <button class="card-action-btn" type="button" on:click={() => { openManageLinks(res.id); }}>
                          Manage
                        </button>
                      {/if}
                      {#if $isAuthenticated}
                        <button class="card-action-btn card-action-retranscode" type="button" on:click={() => { handleRetranscode(res.id); }}>
                          Re-transcode
                        </button>
                        {#if !res.banned}
                          <button class="card-action-btn card-action-ban" type="button" on:click={() => { handleBan(res.id); }}>
                            Ban
                          </button>
                        {/if}
                        <button class="card-action-btn card-action-delete" type="button" on:click={() => { handleDelete(res.id); }}>
                          Delete
                        </button>
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            {/if}

            <!-- Pagination -->
            <div class="mt-4 flex items-center gap-2 text-sm">
              <span class="text-gray-500">{offset + 1}&ndash;{offset + resources.length} of {total}</span>
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
        </div>

        <!-- Upload form -->
        {#if $isAuthenticated}
        <div class="rounded-lg border border-gray-200 bg-white p-4">
          <div
            class="upload-form"
            class:dragging={dragOver}
            role="application"
            on:dragover|preventDefault={() => dragOver = true}
            on:dragleave={() => dragOver = false}
            on:drop|preventDefault={handleDrop}
          >
          <form on:submit|preventDefault={handleUpload}>
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
                  <option value={cat.name}>
                    {cat.display_name || cat.name}{cat.name === 'global' ? ' (public)' : ''}
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
            {#if uploadError}
              <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{uploadError}</div>
            {/if}
            <button type="submit" disabled={uploading || !selectedFile || !uploadForm.title.trim() || !uploadForm.category_id} class="upload-submit-btn">
              {uploading ? 'Uploading…' : 'upload'}
            </button>
          </form>
          </div>
        </div>
        {/if}

      {:else if activeTab === 'history'}
        <TabHistory />

      {:else if activeTab === 'users' && isAdmin}
        <div class="users-section">
          <Users onError={setError} />
        </div>

      {:else if activeTab === 'categories'}
        <Categories onError={setError} />

      {:else if activeTab === 'playlists'}
        <Playlists onError={setError} />
      {/if}

      {#if showCreateLink}
        <!-- Overlay -->
        <div class="fixed inset-0 bg-black/30 z-40" on:click={closeCreateLink}></div>
        <!-- Modal -->
        <div class="fixed inset-0 flex items-center justify-center z-50">
          <div class="rounded-lg border border-gray-200 bg-white p-6 max-w-md w-full shadow-xl mx-4">
            <h3 class="text-base font-semibold text-gray-900 mb-3">Create Share Link</h3>
            
            {#if createLinkUrl}
              <div class="space-y-3">
                <div class="rounded-md bg-green-50 border border-green-200 px-3 py-2 text-sm text-green-700">
                  Link created! Copied to clipboard.
                </div>
                <div class="text-sm text-gray-500 break-all">{window.location.origin}{createLinkUrl}</div>
                <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50" type="button" on:click={closeCreateLink}>Close</button>
              </div>
            {:else}
              <div class="space-y-3">
                <label class="block text-sm font-medium text-gray-700 mb-1">Link expires after</label>

                <div class="flex gap-4 mb-2">
                  <label class="inline-flex items-center gap-1.5 text-sm">
                    <input type="radio" name="expiryMode" value="preset" bind:group={createLinkExpiryMode} on:change={onExpiryModeChange} />
                    Preset
                  </label>
                  <label class="inline-flex items-center gap-1.5 text-sm">
                    <input type="radio" name="expiryMode" value="custom" bind:group={createLinkExpiryMode} on:change={onExpiryModeChange} />
                    Custom
                  </label>
                </div>

                {#if createLinkExpiryMode === 'preset'}
                  <select bind:value={createLinkExpiry} class="w-full">
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
                    <input type="number" bind:value={createLinkCustomMinutes} min={1} max={525600} class="w-full" />
                    <p class="text-xs text-gray-500 mt-1">Minutes (1–525600)</p>
                    <p class="text-sm text-gray-700 mt-1">{formatDuration(createLinkCustomMinutes)}</p>
                  </div>
                {/if}
                
                <p class="text-xs text-gray-400">{formatDuration(createLinkExpiry)}</p>
                
                {#if createLinkError}
                  <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{createLinkError}</div>
                {/if}
                
                <div class="flex gap-2 justify-end">
                  <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50" type="button" on:click={closeCreateLink}>Cancel</button>
                  <button class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50" type="button" disabled={createLinkLoading} on:click={handleCreateLink}>
                    {createLinkLoading ? 'Creating…' : 'Create Link'}
                  </button>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}

      {#if showManageLinks}
        <!-- Overlay -->
        <div class="fixed inset-0 bg-black/30 z-40" on:click={closeManageLinks}></div>
        <!-- Modal -->
        <div class="fixed inset-0 flex items-center justify-center z-50">
          <div class="rounded-lg border border-gray-200 bg-white p-6 max-w-lg w-full shadow-xl mx-4">
            <h3 class="text-base font-semibold text-gray-900 mb-3">Share Links for {manageLinkResourceId}</h3>
            
            {#if manageLinksError}
              <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700 mb-3">{manageLinksError}</div>
            {/if}

            {#if manageLinksLoading}
              <p class="text-sm text-gray-500 mb-4">Loading share links…</p>
            {:else if existingLinks.length === 0}
              <p class="text-sm text-gray-500 mb-4">No share links created yet.</p>
            {:else}
              <div class="space-y-2 mb-4 max-h-60 overflow-y-auto">
                {#each existingLinks as link}
                  <div class="flex items-center justify-between rounded-md border border-gray-200 px-3 py-2 text-sm">
                    <div class="flex-1 min-w-0">
                      <div class="text-gray-900 font-mono truncate">{link.password.slice(0, 8)}...</div>
                      <div class="text-gray-500">Expires: {formatDateTime(link.expires_at)}</div>
                      <div class="text-gray-500">Created: {formatDateTime(link.created_at)}</div>
                    </div>
                    <button
                      class="inline-flex items-center px-2 py-1 border border-red-300 rounded-md text-xs text-red-600 bg-white hover:bg-red-50 disabled:opacity-50 ml-3 flex-shrink-0"
                      type="button"
                      disabled={deleteLinkId === link.password}
                      on:click={() => { handleDeleteLink(link.password); }}
                    >
                      {deleteLinkId === link.password ? 'Deleting…' : 'Cancel'}
                    </button>
                  </div>
                {/each}
              </div>
            {/if}

            <div class="flex justify-end border-t border-gray-100 pt-3">
              <button
                class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50"
                type="button"
                on:click={closeManageLinks}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      {/if}

      {#if showLoginModal}
        <!-- Overlay -->
        <div class="fixed inset-0 bg-black/30 z-40" on:click={() => { showLoginModal = false; }}></div>
        <!-- Modal -->
        <div class="fixed inset-0 flex items-center justify-center z-50">
          <div class="rounded-lg border border-gray-200 bg-white p-6 max-w-sm w-full shadow-xl mx-4">
            <div class="flex justify-between items-center mb-4">
              <h2 class="text-lg font-semibold text-gray-900">Login</h2>
              <button class="text-gray-400 hover:text-gray-600 text-xl leading-none" on:click={() => { showLoginModal = false; }}>&times;</button>
            </div>
            <Login onSuccess={handleLoginSuccess} />
          </div>
        </div>
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

  .action-bar-left select {
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

  .resource-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 0.75rem;
  }

  .resource-card {
    border: 1px solid #e5e7eb;
    border-radius: 0.5rem;
    background: white;
    padding: 0.75rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    position: relative;
  }

  .resource-card:hover {
    border-color: #d1d5db;
    box-shadow: 0 1px 3px rgba(0,0,0,0.06);
  }

  .card-checkbox {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
  }

  .card-title {
    font-size: 0.9rem;
    line-height: 1.3;
    padding-right: 1.5rem;
  }

  .card-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
    font-size: 0.8rem;
    color: #6b7280;
  }

  .card-category {
    font-size: 0.75rem;
    color: #6b7280;
  }

  .card-stats {
    display: flex;
    gap: 1rem;
    font-size: 0.75rem;
    color: #9ca3af;
  }

  .card-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    padding-top: 0.25rem;
    border-top: 1px solid #f3f4f6;
  }

  .card-action-btn {
    padding: 0.15rem 0.45rem;
    font-size: 0.7rem;
    border: 1px solid #d1d5db;
    border-radius: 0.25rem;
    background: white;
    color: #374151;
    cursor: pointer;
    white-space: nowrap;
  }

  .card-action-btn:hover {
    background: #f3f4f6;
  }

  .card-action-retranscode {
    color: #6366f1;
    border-color: #c7d2fe;
  }

  .card-action-retranscode:hover {
    background: #eef2ff;
  }

  .card-action-ban {
    color: #dc2626;
    border-color: #fecaca;
  }

  .card-action-ban:hover {
    background: #fef2f2;
  }

  .card-action-delete {
    color: #dc2626;
    border-color: #fecaca;
  }

  .card-action-delete:hover {
    background: #fef2f2;
  }

  .view-toggle-btn {
    padding: 0.25rem 0.6rem;
    font-size: 0.8rem;
    border: 1px solid #d1d5db;
    background: white;
    color: #374151;
    cursor: pointer;
    white-space: nowrap;
  }
  .view-toggle-btn:first-child {
    border-radius: 0.25rem 0 0 0.25rem;
  }
  .view-toggle-btn:last-child {
    border-radius: 0 0.25rem 0.25rem 0;
    border-left: none;
  }
  .view-toggle-active {
    background: #6366f1;
    color: white;
    border-color: #6366f1;
  }
  .view-toggle-active + .view-toggle-btn {
    border-left: 1px solid #6366f1;
  }

  @media (max-width: 768px) {
    .content-row {
      flex-direction: column;
    }
    .resource-grid {
      grid-template-columns: 1fr;
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

  .upload-form {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .upload-form.dragging {
    outline: 2px dashed #6366f1;
    outline-offset: 4px;
    border-radius: 0.375rem;
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
    margin-left: auto;
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
    overflow: hidden;
    text-overflow: ellipsis;
    flex: 1;
    min-width: 0;
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

  .user-section {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-left: auto;
  }

  .username-label {
    font-size: 0.85rem;
    color: #6b7280;
  }

  .login-btn {
    padding: 0.25rem 0.75rem;
    font-size: 0.85rem;
    border: 1px solid #d1d5db;
    border-radius: 0.375rem;
    background: white;
    color: #374151;
    cursor: pointer;
    white-space: nowrap;
  }

  .login-btn:hover {
    background: #f3f4f6;
  }
</style>
