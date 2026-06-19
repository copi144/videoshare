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

<div class="space-y-4">
  {#if createdUser}
    <div class="rounded-lg border border-gray-200 bg-white p-4">
      <h2 class="text-base font-semibold text-gray-900 mb-1">User Created: {createdUser.username}</h2>
      <p class="text-sm text-gray-500 mb-3">Share the following TOTP setup information with the new user. This information will not be shown again.</p>
      {#if createdUser.qr_image}
        <div class="mb-3">
          <img src={createdUser.qr_image} alt="TOTP QR Code for {createdUser.username}" class="rounded-lg border border-gray-200" style="max-width: 200px;" />
          <p class="text-xs text-gray-400 mt-1">Scan with your authenticator app</p>
        </div>
      {/if}
      <div class="space-y-2">
        <div>
          <label for="totp_secret" class="block text-xs font-medium text-gray-500 mb-1">Secret</label>
          <input type="text" id="totp_secret" value={createdUser.totp_secret} readonly class="w-full bg-gray-50 text-sm" />
        </div>
        <div>
          <label for="totp_uri" class="block text-xs font-medium text-gray-500 mb-1">TOTP URI</label>
          <input type="text" id="totp_uri" value={createdUser.totp_uri} readonly class="w-full bg-gray-50 text-sm" />
        </div>
      </div>
    </div>
  {/if}

  <!-- Create form -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Create User</h2>
    <form on:submit|preventDefault={handleCreate} class="flex items-end gap-3">
      <div class="flex-1">
        <label for="username" class="block text-sm font-medium text-gray-700 mb-1">Username</label>
        <input type="text" id="username" name="username" bind:value={username} required autocomplete="off" pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" class="w-full" />
      </div>
      <button type="submit" disabled={loading} aria-busy={loading} class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 whitespace-nowrap">
        {loading ? 'Creating…' : 'Create User'}
      </button>
    </form>
  </div>
</div>
