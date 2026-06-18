<script lang="ts">
  import { shareAuth } from '../lib/api';

  export let id: string;
  export let onSuccess: (id: string) => void;

  let password = '';
  let error: string | null = null;
  let loading = false;

  async function handleSubmit() {
    error = null;
    loading = true;
    try {
      const result = await shareAuth(id, password);
      if (result.ok) {
        onSuccess(id);
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
