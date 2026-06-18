<script lang="ts">
  import { onMount } from 'svelte';
  import { page, navigate, checkAuth } from './stores/auth';
  
  // Lazy page imports
  let PageComponent: any = null;
  
  $: {
    // Resolve page based on current route
    const p = $page;
    if (!p) {
      PageComponent = null;
    } else if (p.name === 'loading') {
      import('./pages/Loading.svelte').then(m => PageComponent = m.default);
    } else if (p.name === 'login') {
      import('./pages/Login.svelte').then(m => PageComponent = m.default);
    } else if (p.name === 'admin') {
      import('./pages/Admin.svelte').then(m => PageComponent = m.default);
    } else if (p.name === 'share') {
      import('./pages/Share.svelte').then(m => PageComponent = m.default);
    } else if (p.name === 'watch') {
      import('./pages/Watch.svelte').then(m => PageComponent = m.default);
    } else if (p.name === 'categories') {
      import('./pages/Categories.svelte').then(m => PageComponent = m.default);
    } else if (p.name === 'playlists') {
      import('./pages/Playlists.svelte').then(m => PageComponent = m.default);
    } else if (p.name === 'users') {
      import('./pages/Users.svelte').then(m => PageComponent = m.default);
    } else {
      import('./pages/NotFound.svelte').then(m => PageComponent = m.default);
    }
  }
  
  onMount(async () => {
    // Listen for hash changes
    function handleHash() {
      const hash = window.location.hash.slice(1) || '/';
      navigate(hash);
    }
    window.addEventListener('hashchange', handleHash);
    
    // Check auth state on load
    await checkAuth();
    
    // Parse initial route
    handleHash();
    
    return () => window.removeEventListener('hashchange', handleHash);
  });
</script>

<main class="container">
  {#if PageComponent}
    <svelte:component this={PageComponent} params={$page.params} />
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
