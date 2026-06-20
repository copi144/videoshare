<script lang="ts">
  import { onMount } from 'svelte';
  import { getShareLinkResources } from '../lib/api';

  export let id: string;
  export let password: string;

  let loading = true;
  let error: string | null = null;
  let targetType: string | null = null;
  let targetName: string | null = null;
  let resources: Array<{
    id: string;
    title: string;
    filename: string;
    file_size: number;
    content_type: string;
    resource_type: string;
    views: number;
    created_at: string;
  }> = [];

  onMount(async () => {
    if (!id || !password) {
      error = 'Invalid share link.';
      loading = false;
      return;
    }

    try {
      const result = await getShareLinkResources(id, password);
      targetType = result.target_type;
      targetName = result.target_name;
      resources = result.resources || [];
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load shared content.';
    } finally {
      loading = false;
    }
  });

  function formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
    return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
  }

  function getDownloadUrl(res: { id: string; resource_type: string }): string {
    if (res.resource_type === 'video') return `/v/${res.id}/download`;
    if (res.resource_type === 'audio') return `/a/${res.id}`;
    if (res.resource_type === 'image') return `/i/${res.id}`;
    return '#';
  }

  function getTypeIcon(type: string): string {
    if (type === 'video') return '🎬';
    if (type === 'audio') return '🎵';
    if (type === 'image') return '🖼️';
    return '📄';
  }
</script>

<div class="max-w-4xl mx-auto mt-8 px-4">

  {#if loading}
    <p class="text-gray-500 text-sm">Loading shared content…</p>
  {:else if error}
    <div class="rounded-lg border border-red-200 bg-red-50 p-6">
      <h2 class="text-lg font-semibold text-red-800 mb-2">Access Denied</h2>
      <p class="text-sm text-red-600">{error}</p>
      <p class="text-xs text-red-500 mt-2">The share link may be invalid, expired, or the password is incorrect.</p>
    </div>
  {:else}
    <div class="rounded-lg border border-gray-200 bg-white p-6 mb-4">
      <h2 class="text-lg font-semibold text-gray-900">{(targetType === 'playlist' ? '📋' : '📁')} {targetName}</h2>
      <p class="text-sm text-gray-500 mt-1">{resources.length} file{resources.length !== 1 ? 's' : ''} shared</p>
    </div>

    {#if resources.length === 0}
      <p class="text-gray-500 text-sm">No files in this {targetType}.</p>
    {:else}
      <div class="rounded-lg border border-gray-200 bg-white overflow-hidden">
        <table class="w-full text-left text-sm">
          <thead>
            <tr class="border-b border-gray-200 bg-gray-50">
              <th class="py-2 px-4 text-xs font-medium text-gray-500 uppercase">Type</th>
              <th class="py-2 px-4 text-xs font-medium text-gray-500 uppercase">Title</th>
              <th class="py-2 px-4 text-xs font-medium text-gray-500 uppercase">Size</th>
              <th class="py-2 px-4 text-xs font-medium text-gray-500 uppercase">Views</th>
              <th class="py-2 px-4 text-xs font-medium text-gray-500 uppercase">Date</th>
              <th class="py-2 px-4 text-xs font-medium text-gray-500 uppercase">Link</th>
            </tr>
          </thead>
          <tbody>
            {#each resources as res}
              <tr class="border-b border-gray-100 hover:bg-gray-50">
                <td class="py-2 px-4 text-lg">{getTypeIcon(res.resource_type)}</td>
                <td class="py-2 px-4 font-medium text-gray-900">{res.title}</td>
                <td class="py-2 px-4 text-gray-500">{formatFileSize(res.file_size)}</td>
                <td class="py-2 px-4 text-gray-500">{res.views}</td>
                <td class="py-2 px-4 text-gray-500">{new Date(res.created_at).toLocaleDateString()}</td>
                <td class="py-2 px-4">
                  <a
                    href={getDownloadUrl(res)}
                    target="_blank"
                    class="inline-flex items-center px-3 py-1 rounded-md text-xs font-medium text-white bg-indigo-600 hover:bg-indigo-700"
                  >
                    {res.resource_type === 'video' ? 'Download' : res.resource_type === 'audio' ? 'Listen' : 'View'}
                  </a>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {/if}
</div>
