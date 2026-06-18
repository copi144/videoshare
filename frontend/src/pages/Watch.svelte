<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { getResource } from '../lib/api';
  import { addWatchHistory } from '../stores/history';
  import Hls from 'hls.js';

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
    transcode_status?: string;
  }

  let resource: ResourceInfo | null = null;
  let error: string | null = null;
  let loading = true;
  let videoRef: HTMLVideoElement;
  let hlsInstance: Hls | null = null;

  function formatSize(bytes: number): string {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  }

  // Initialize HLS or native playback when resource and video element are ready
  $: if (resource && videoRef) {
    const videoId = params.id;
    // Clean up previous HLS instance if any
    if (hlsInstance) {
      hlsInstance.destroy();
      hlsInstance = null;
    }
    if (resource.transcode_status === 'done') {
      if (Hls.isSupported()) {
        hlsInstance = new Hls();
        hlsInstance.loadSource(`/api/video/${videoId}/hls/master.m3u8`);
        hlsInstance.attachMedia(videoRef);
      } else if (videoRef.canPlayType('application/vnd.apple.mpegurl')) {
        // Native HLS support (Safari)
        videoRef.src = `/api/video/${videoId}/hls/master.m3u8`;
      }
    } else {
      // Fall back to original file
      videoRef.src = `/api/video/${videoId}`;
    }
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

  onDestroy(() => {
    if (hlsInstance) {
      hlsInstance.destroy();
      hlsInstance = null;
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
    <video controls bind:this={videoRef} style="width: 100%; max-height: 80vh;">
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
