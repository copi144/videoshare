<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { checkAuth, isAuthenticated, startHeartbeat } from './stores/auth';
  import Login from './pages/Login.svelte';
  import MainApp from './pages/MainApp.svelte';
  import Share from './pages/Share.svelte';
  import Watch from './pages/Watch.svelte';

  type View = 'login' | 'admin' | 'share' | 'watch';

  let view: View = 'login';
  let shareId: string | null = null;

  function handleLoginSuccess() {
    view = 'admin';
    startHeartbeat();
  }

  function handleShareSuccess(id: string) {
    shareId = id;
    view = 'watch';
  }

  // Extract share ID from hash and navigate to share/watch view
  function handleHashChange() {
    const match = window.location.hash.match(/^#\/v\/([^/]+)(?:\/watch)?\/?$/);
    if (match) {
      shareId = match[1];
      view = 'share';
    }
  }

  onMount(async () => {
    // Check if this is a share URL
    const match = window.location.hash.match(/^#\/v\/([^/]+)(?:\/watch)?\/?$/);
    if (match) {
      shareId = match[1];
      view = 'share';
    }

    // Check auth state
    await checkAuth();

    // If authenticated (not on a share page), start heartbeat
    if ($isAuthenticated && !match) {
      view = 'admin';
      startHeartbeat();
    } else if (!match) {
      view = 'login';
    }

    // Rewrite non-share URLs to / so only / is visible in the address bar
    if (!match) {
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
  {#if view === 'login'}
    <Login onSuccess={handleLoginSuccess} />
  {:else if view === 'admin'}
    <MainApp />
  {:else if view === 'share' && shareId}
    <Share id={shareId} onSuccess={handleShareSuccess} />
  {:else if view === 'watch' && shareId}
    <Watch id={shareId} />
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
  }
</style>
