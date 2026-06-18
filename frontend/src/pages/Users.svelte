<script lang="ts">
  import { createUser } from '../lib/api';

  export let onError: ((msg: string) => void) | undefined = undefined;

  let username = '';
  let loading = false;

  interface CreatedUser {
    username: string;
    totp_secret: string;
    totp_uri: string;
    qr_image: string;
  }

  let createdUser: CreatedUser | null = null;

  async function handleCreate() {
    createdUser = null;

    if (!username.trim()) {
      onError?.('Username is required.');
      return;
    }

    loading = true;
    try {
      const result = await createUser(username.trim());
      if (result.ok) {
        createdUser = {
          username: username.trim(),
          totp_secret: result.totp_secret,
          totp_uri: result.totp_uri,
          qr_image: result.qr_image,
        };
        username = '';
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create user.';
      onError?.(msg);
    } finally {
      loading = false;
    }
  }
</script>

<h1>User Management</h1>

<article>
  <h2>Create User</h2>
  <form on:submit|preventDefault={handleCreate}>
    <label for="username">
      Username
      <input type="text" id="username" name="username" bind:value={username} required autocomplete="off" pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" />
    </label>
    <button type="submit" disabled={loading} aria-busy={loading}>
      {loading ? 'Creating…' : 'Create User'}
    </button>
  </form>
</article>

{#if createdUser}
  <article>
    <hgroup>
      <h2>User Created: {createdUser.username}</h2>
      <p>Share the following TOTP setup information with the new user. This information will not be shown again.</p>
    </hgroup>
    {#if createdUser.qr_image}
      <figure>
        <img src={createdUser.qr_image} alt="TOTP QR Code for {createdUser.username}" style="max-width: 256px;" />
        <figcaption>Scan this QR code with your authenticator app</figcaption>
      </figure>
    {/if}
    <label for="totp_secret">
      Secret
      <input type="text" id="totp_secret" value={createdUser.totp_secret} readonly />
    </label>
    <label for="totp_uri">
      TOTP URI
      <input type="text" id="totp_uri" value={createdUser.totp_uri} readonly />
    </label>
  </article>
{/if}
