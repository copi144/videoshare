<script lang="ts">
  import { login } from '../lib/api';

  export let onSuccess: () => void;

  let username = '';
  let totpCode = '';
  let error: string | null = null;
  let loading = false;

  async function handleSubmit() {
    error = null;
    loading = true;
    try {
      const result = await login(username, totpCode);
      if (result.ok) {
        onSuccess();
      } else {
        error = 'Login failed. Please check your credentials.';
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Login failed. Please check your credentials.';
    } finally {
      loading = false;
    }
  }
</script>

<h1>Login</h1>
<article>
  <form on:submit|preventDefault={handleSubmit}>
    <label for="username">
      Username
      <input type="text" id="username" name="username" bind:value={username} required autocomplete="username" pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" />
    </label>
    <label for="totp_code">
      Authentication Code
      <input
        type="text"
        id="totp_code"
        name="totp_code"
        pattern={"[0-9]{6}"}
        inputmode="numeric"
        maxlength={6}
        bind:value={totpCode}
        required
        autocomplete="one-time-code"
      />
    </label>
    {#if error}
      <article class="error-box">{error}</article>
    {/if}
    <button type="submit" disabled={loading} aria-busy={loading}>
      {loading ? 'Logging in…' : 'Login'}
    </button>
  </form>
</article>
