<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { getResource } from '../lib/api';
  import { addWatchHistory } from '../stores/history';
  import Hls from 'hls.js';

  export let id: string;

  interface ResourceInfo {
    id: string;
    title: string;
    readme?: string;
    content_type: string;
    resource_type?: string;
    file_size: number;
    views: number;
    created_at: string;
    updated_at?: string;
    uploaded_by: string;
    uploaded_username: string;
    filename?: string;
    category_name: string;
    transcode_status?: string;
    banned?: boolean;
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
    const videoId = id;
    const rtype = resource.resource_type || 'video';
    if (rtype === 'video') {
      // Clean up previous HLS instance if any
      if (hlsInstance) {
        hlsInstance.destroy();
        hlsInstance = null;
      }
      if (resource.transcode_status === 'done') {
        if (Hls.isSupported()) {
          hlsInstance = new Hls();
          hlsInstance.loadSource(`/v/${videoId}/hls/master.m3u8`);
          hlsInstance.attachMedia(videoRef);
        } else if (videoRef.canPlayType('application/vnd.apple.mpegurl')) {
          // Native HLS support (Safari)
          videoRef.src = `/v/${videoId}/hls/master.m3u8`;
        }
      } else {
        // Fall back to original file
        videoRef.src = `/v/${videoId}`;
      }
    }
  }

  onMount(async () => {
    if (!id) {
      error = 'No video ID specified.';
      loading = false;
      return;
    }
    try {
      resource = await getResource(id);
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
  <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">{error}</div>
{:else if resource.banned}
  <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700">This video has been banned.</div>
{:else if resource}
  <h2>{resource.title}</h2>
  {#if resource.resource_type === 'audio'}
    <div class="rounded-lg border border-gray-200 bg-white p-4">
      <audio controls class="w-full" bind:this={videoRef}>
        <source src="/a/{resource.id}" type={resource.content_type} />
        Your browser does not support the audio element.
      </audio>
    </div>
  {:else if resource.resource_type === 'image'}
    <div class="rounded-lg border border-gray-200 bg-white p-4" style="text-align: center;">
      <img src="/i/{resource.id}" alt={resource.title} style="max-width: 100%; max-height: 80vh;" />
    </div>
  {:else}
    <div class="rounded-lg border border-gray-200 bg-white p-4">
      <video controls bind:this={videoRef} style="width: 100%; max-height: 80vh;">
        <track kind="captions" label="No captions available" />
        Your browser does not support the video tag.
      </video>
    </div>
  {/if}
  {#if resource.readme}
    <div class="rounded-lg border border-gray-200 bg-white p-4">
      <pre style="white-space: pre-wrap; margin: 0; font-family: inherit;">{resource.readme}</pre>
    </div>
  {:else}
    <p class="text-gray-400 italic">No description</p>
  {/if}
  <p>Views: {resource.views} | Size: {formatSize(resource.file_size)} | Uploaded by: {resource.uploaded_username}</p>
  {#if resource.resource_type === 'audio'}
    <a href="/a/{resource.id}" class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50 no-underline">Download Original</a>
  {:else if resource.resource_type === 'image'}
    <a href="/i/{resource.id}" class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50 no-underline">Download Original</a>
  {:else}
    <a href="/v/{resource.id}/download" class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50 no-underline">Download Original</a>
  {/if}
{/if}
