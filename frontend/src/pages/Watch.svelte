<script lang="ts">
  import { onMount } from 'svelte';
  import { getResource } from '../lib/api';
  import { addWatchHistory } from '../stores/history';

  export let params: Record<string, string> = {};

  interface ResourceInfo {
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
      const data = await getResource(params.id);
      resource = data.resource;
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
  <p>{resource.description}</p>
  <p>Views: {resource.views} | Size: {formatSize(resource.file_size)} | Uploaded by: {resource.uploaded_username}</p>
  <a href="#/admin">← Back to Videos</a>
{/if}
