<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import MainApp from './pages/MainApp.svelte';
  import Share from './pages/Share.svelte';
  import Watch from './pages/Watch.svelte';
  import ShareLink from './pages/ShareLink.svelte';

  type View = 'admin' | 'share' | 'watch' | 'share-link';

  let view: View = 'admin';
  let shareId: string | null = null;
  let sharePassword: string | null = null;
  let shareLinkId: string | null = null;
  let shareLinkPassword: string | null = null;

  function handleShareSuccess(id: string, password?: string) {
    shareId = id;
    sharePassword = password || null;
    view = 'watch';
  }

  // Extract share ID from hash and navigate to share/watch view
  function handleHashChange() {
    // /#/v/{hash}/{password} — video share
    const videoMatch = window.location.hash.match(/^#\/([vai])\/([^/]+?)(?:\/([a-f0-9]+))?(?:\/watch)?\/?$/);
    if (videoMatch) {
      shareId = videoMatch[2];
      sharePassword = videoMatch[3] || null;
      view = 'share';
      return;
    }
    // /#/s/{id}/{password} — category/playlist share
    const shareLinkMatch = window.location.hash.match(/^#\/s\/([a-f0-9]+)\/([a-f0-9]+)\/?$/);
    if (shareLinkMatch) {
      shareLinkId = shareLinkMatch[1];
      shareLinkPassword = shareLinkMatch[2];
      view = 'share-link';
      return;
    }
    // Default: show MainApp
    view = 'admin';
  }

  onMount(async () => {
    // Check if this is a share URL — video share
    const videoMatch = window.location.hash.match(/^#\/([vai])\/([^/]+?)(?:\/([a-f0-9]+))?(?:\/watch)?\/?$/);
    // Check if this is a category/playlist share URL
    const shareLinkMatch = window.location.hash.match(/^#\/s\/([a-f0-9]+)\/([a-f0-9]+)\/?$/);

    if (videoMatch) {
      shareId = videoMatch[2];
      sharePassword = videoMatch[3] || null;
      view = 'share';
    } else if (shareLinkMatch) {
      shareLinkId = shareLinkMatch[1];
      shareLinkPassword = shareLinkMatch[2];
      view = 'share-link';
    } else {
      view = 'admin';
    }

    // Rewrite non-share URLs to / so only / is visible in the address bar
    if (!videoMatch && !shareLinkMatch) {
      history.replaceState(null, '', '/');
    }

    // Listen for hash changes (internal navigation via links)
    window.addEventListener('hashchange', handleHashChange);
  });

  onDestroy(() => {
    window.removeEventListener('hashchange', handleHashChange);
  });
</script>

<main class="mx-auto px-4 sm:px-6">
  {#if view === 'admin'}
    <MainApp />
  {:else if view === 'share' && shareId}
    <Share id={shareId} password={sharePassword || ''} onSuccess={handleShareSuccess} />
  {:else if view === 'watch' && shareId}
    <Watch id={shareId} />
  {:else if view === 'share-link' && shareLinkId && shareLinkPassword}
    <ShareLink id={shareLinkId} password={shareLinkPassword} />
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
  }
</style>
