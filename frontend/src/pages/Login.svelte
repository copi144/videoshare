<script lang="ts">
  import { createSession, setApiToken } from '../lib/api';
  import { apiToken } from '../stores/auth';

  export let onSuccess: () => void;

  let name = '';
  let totpCode = '';
  let error: string | null = null;
  let loading = false;

  // Name validation pattern
  const namePattern = /^[0-9A-Za-z\-]*$/;
  $: nameValid = name === '' || namePattern.test(name);

  async function handleSubmit() {
    error = null;
    loading = true;
    try {
      const result = await createSession('user', { name, totp_code: totpCode });
      if (result.ok) {
        if (result.api_token) {
          setApiToken(result.api_token);
          apiToken.set(result.api_token);
        }
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

<div>
  <form on:submit|preventDefault={handleSubmit}>
    <label for="name">
      Name
      <input type="text" id="name" name="name" bind:value={name} required autocomplete="username" pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" class:border-red-500={name !== '' && !nameValid} />
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
      <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700 mb-4">{error}</div>
    {/if}
    <button type="submit" disabled={loading} aria-busy={loading}>
      {loading ? 'Logging in…' : 'Login'}
    </button>
  </form>
</div>
