<script lang="ts">
  import { onMount } from 'svelte';
  import { checkAuth, isAuthenticated, startHeartbeat } from './stores/auth';
  import Login from './pages/Login.svelte';
  import Admin from './pages/Admin.svelte';
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

  onMount(async () => {
    // Check if this is a share URL
    const match = window.location.pathname.match(/^\/s\/([^/]+)(?:\/watch)?\/?$/);
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
  });
</script>

<main class="container">
  {#if view === 'login'}
    <Login onSuccess={handleLoginSuccess} />
  {:else if view === 'admin'}
    <Admin />
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
  :global(main.container) {
    padding-top: 1rem;
    padding-bottom: 2rem;
  }
</style>
