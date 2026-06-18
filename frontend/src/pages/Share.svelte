<script lang="ts">
  import { navigate } from '../stores/auth';
  import { shareAuth } from '../lib/api';

  export let params: Record<string, string> = {};

  let password = '';
  let error: string | null = null;
  let loading = false;

  async function handleSubmit() {
    if (!params.id) {
      error = 'Invalid share link.';
      return;
    }
    error = null;
    loading = true;
    try {
      const result = await shareAuth(params.id, password);
      if (result.ok) {
        const target = result.redirect || `/s/${params.id}/watch`;
        navigate(target);
      } else {
        error = 'Incorrect password.';
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Authentication failed.';
    } finally {
      loading = false;
    }
  }
</script>

<h1>Enter Video Password</h1>
<article>
  <form on:submit|preventDefault={handleSubmit}>
    <label for="password">
      Password
      <input type="password" id="password" name="password" bind:value={password} required autocomplete="off" />
    </label>
    {#if error}
      <article class="error-box">{error}</article>
    {/if}
    <button type="submit" disabled={loading} aria-busy={loading}>
      {loading ? 'Authenticating…' : 'Access Video'}
    </button>
  </form>
</article>
