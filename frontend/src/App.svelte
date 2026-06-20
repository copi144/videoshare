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
  let initialCategory: string | null = null;
  let initialPlaylist: string | null = null;

  function handleShareSuccess(id: string, password?: string) {
    shareId = id;
    sharePassword = password || null;
    view = 'watch';
  }

  function handleHashChange() {
    // /#/v/{hash}/{password} — video share
    const videoMatch = window.location.hash.match(/^#\/([vai])\/([^/]+?)(?:\/([a-f0-9]+))?(?:\/watch)?\/?$/);
    if (videoMatch) {
      shareId = videoMatch[2];
      sharePassword = videoMatch[3] || null;
      view = 'share';
      initialCategory = null;
      initialPlaylist = null;
      return;
    }
    // /#/s/{id}/{password} — category/playlist share
    const shareLinkMatch = window.location.hash.match(/^#\/s\/([a-f0-9]+)\/([a-f0-9]+)\/?$/);
    if (shareLinkMatch) {
      shareLinkId = shareLinkMatch[1];
      shareLinkPassword = shareLinkMatch[2];
      view = 'share-link';
      initialCategory = null;
      initialPlaylist = null;
      return;
    }
    // /#/c/{name} — navigate to browse with category filter
    const catMatch = window.location.hash.match(/^#\/c\/([^/]+?)\/?$/);
    if (catMatch) {
      initialCategory = catMatch[1];
      initialPlaylist = null;
      view = 'admin';
      return;
    }
    // /#/l/{category}/{name} — navigate to browse with playlist filter
    const plMatch = window.location.hash.match(/^#\/l\/([^/]+)\/([^/]+?)\/?$/);
    if (plMatch) {
      initialPlaylist = plMatch[1] + ':' + plMatch[2]; // composite "category:name"
      initialCategory = null;
      view = 'admin';
      return;
    }
    // Default: show MainApp
    initialCategory = null;
    initialPlaylist = null;
    view = 'admin';
  }

  onMount(async () => {
    const videoMatch = window.location.hash.match(/^#\/([vai])\/([^/]+?)(?:\/([a-f0-9]+))?(?:\/watch)?\/?$/);
    const shareLinkMatch = window.location.hash.match(/^#\/s\/([a-f0-9]+)\/([a-f0-9]+)\/?$/);
    const catMatch = window.location.hash.match(/^#\/c\/([^/]+?)\/?$/);
    const plMatch = window.location.hash.match(/^#\/l\/([^/]+)\/([^/]+?)\/?$/);

    if (videoMatch) {
      shareId = videoMatch[2];
      sharePassword = videoMatch[3] || null;
      view = 'share';
    } else if (shareLinkMatch) {
      shareLinkId = shareLinkMatch[1];
      shareLinkPassword = shareLinkMatch[2];
      view = 'share-link';
    } else if (catMatch) {
      initialCategory = catMatch[1];
      view = 'admin';
    } else if (plMatch) {
      initialPlaylist = plMatch[1] + ':' + plMatch[2];
      view = 'admin';
    } else {
      view = 'admin';
    }

    // Rewrite non-share URLs to / so only / is visible in the address bar
    if (!videoMatch && !shareLinkMatch) {
      history.replaceState(null, '', '/');
    }

    window.addEventListener('hashchange', handleHashChange);
  });

  onDestroy(() => {
    window.removeEventListener('hashchange', handleHashChange);
  });
</script>

<main class="mx-auto px-4 sm:px-6">
  {#if view === 'admin'}
    <MainApp {initialCategory} {initialPlaylist} />
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
