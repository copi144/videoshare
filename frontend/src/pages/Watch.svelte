<script lang="ts">
  import { onMount } from 'svelte';
  import { getResource } from '../lib/api';
  import { addWatchHistory } from '../stores/history';

  export let params: Record<string, string> = {};

  interface ResourceInfo {
    id: string;
    title: string;
    readme?: string;
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
  }

  let resource: ResourceInfo | null = null;
  let error: string | null = null;
  let loading = true;

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  onMount(async () => {
    if (!params.id) {
      error = 'No video ID specified.';
      loading = false;
      return;
    }
    try {
      resource = await getResource(params.id);
      addWatchHistory({ id: resource.id, title: resource.title, watchedAt: new Date().toISOString() });
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load video info.';
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <p aria-busy="true">Loading video…</p>
{:else if error}
  <article class="error-box">{error}</article>
{:else if resource}
  <h2>{resource.title}</h2>
  <article>
    <video controls src="/api/video/{params.id}" style="width: 100%; max-height: 80vh;">
      <track kind="captions" label="No captions available" />
      Your browser does not support the video tag.
    </video>
  </article>
  {#if resource.readme}
    <article>
      <pre style="white-space: pre-wrap; margin: 0; font-family: inherit;">{resource.readme}</pre>
    </article>
  {:else}
    <p style="color: var(--muted-color, #888); font-style: italic;">No description</p>
  {/if}
  <p>Views: {resource.views} | Size: {formatSize(resource.file_size)} | Uploaded by: {resource.uploaded_username}</p>
  <a href="#/admin">← Back to Videos</a>
{/if}
