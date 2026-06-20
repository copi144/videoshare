<script lang="ts">
  import { onMount } from 'svelte';
  import { listUsers, createUser, deleteUser, resetTOTP } from '../lib/api';

  export let onError: ((msg: string) => void) | undefined = undefined;

  interface User {
    id: string;
    username: string;
    display_name: string;
    role: string;
    created_at: string;
  }

  let users: User[] = [];
  let loading = true;
  let username = '';
  let displayName = '';
  let role = 'uploader';
  let creating = false;

  let totpResult: { username: string; totp_secret: string; totp_uri: string; qr_image: string } | null = null;

  onMount(loadUsers);

  async function loadUsers() {
    loading = true;
    try {
      const data = await listUsers();
      users = data.users;
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to load users.';
      onError?.(msg);
    } finally {
      loading = false;
    }
  }

  async function handleCreate() {
    totpResult = null;
    if (!username.trim()) {
      onError?.('Username is required.');
      return;
    }
    creating = true;
    try {
      const result = await createUser(username.trim(), role, displayName.trim());
      if (result.ok) {
        totpResult = {
          username: username.trim(),
          totp_secret: result.totp_secret,
          totp_uri: result.totp_uri,
          qr_image: result.qr_image,
        };
        username = '';
        displayName = '';
        role = 'uploader';
        await loadUsers();
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create user.';
      onError?.(msg);
    } finally {
      creating = false;
    }
  }

  async function handleDelete(id: string, username: string) {
    if (username === 'admin') {
      onError?.('Cannot delete the root admin user.');
      return;
    }
    if (!confirm(`Delete user "${username}"? This cannot be undone.`)) return;
    try {
      await deleteUser(id);
      await loadUsers();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to delete user.';
      onError?.(msg);
    }
  }

  async function handleResetTOTP(id: string, username: string) {
    if (!confirm(`Reset TOTP for "${username}"? The user will need to reconfigure their authenticator app.`)) return;
    try {
      const result = await resetTOTP(id);
      if (result.ok) {
        totpResult = {
          username,
          totp_secret: result.totp_secret,
          totp_uri: result.totp_uri,
          qr_image: result.qr_image,
        };
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to reset TOTP.';
      onError?.(msg);
    }
  }
</script>

<div class="space-y-4">
  <!-- TOTP Result -->
  {#if totpResult}
    <div class="rounded-lg border border-gray-200 bg-white p-4">
      <h2 class="text-base font-semibold text-gray-900 mb-1">TOTP Setup: {totpResult.username}</h2>
      <p class="text-sm text-gray-500 mb-3">Share this information with the user. It will not be shown again.</p>
      {#if totpResult.qr_image}
        <div class="mb-3">
          <img src={totpResult.qr_image} alt="TOTP QR Code" class="rounded-lg border" style="max-width: 200px;" />
          <p class="text-xs text-gray-400 mt-1">Scan with your authenticator app</p>
        </div>
      {/if}
      <div class="space-y-2">
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">Secret</label>
          <input type="text" value={totpResult.totp_secret} readonly class="w-full bg-gray-50 text-sm" />
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">TOTP URI</label>
          <input type="text" value={totpResult.totp_uri} readonly class="w-full bg-gray-50 text-sm" />
        </div>
      </div>
    </div>
  {/if}

  <!-- User List -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Users</h2>
    {#if loading}
      <p class="text-gray-500 text-sm">Loading users…</p>
    {:else if users.length === 0}
      <p class="text-gray-500 text-sm">No users yet.</p>
    {:else}
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200">
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Username</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Display Name</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Role</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Created</th>
            <th class="py-2 text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each users as u}
            <tr class="border-b border-gray-100">
              <td class="py-2 pr-4">{u.username}</td>
              <td class="py-2 pr-4 text-gray-500">{u.display_name || '—'}</td>
              <td class="py-2 pr-4">
                <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {u.role === 'admin' ? 'bg-red-100 text-red-700' : u.role === 'uploader' ? 'bg-blue-100 text-blue-700' : 'bg-gray-100 text-gray-700'}">
                  {u.role}
                </span>
              </td>
              <td class="py-2 pr-4 text-gray-500">{new Date(u.created_at).toLocaleDateString()}</td>
              <td class="py-2">
                <button class="row-action-btn" type="button" on:click={() => handleResetTOTP(u.id, u.username)}>Reset TOTP</button>
                <button class="row-action-btn row-action-delete" type="button" on:click={() => handleDelete(u.id, u.username)} disabled={u.username === 'admin'}>Delete</button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <!-- Create form -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Create User</h2>
    <form on:submit|preventDefault={handleCreate} class="space-y-3">
      <div>
        <label for="username" class="block text-sm font-medium text-gray-700 mb-1">Username</label>
        <input type="text" id="username" bind:value={username} required pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" class="w-full" />
      </div>
      <div>
        <label for="display_name" class="block text-sm font-medium text-gray-700 mb-1">Display Name</label>
        <input type="text" id="display_name" bind:value={displayName} class="w-full" />
      </div>
      <div>
        <label for="role" class="block text-sm font-medium text-gray-700 mb-1">Role</label>
        <select id="role" bind:value={role} class="w-full">
          <option value="uploader">Uploader (can upload to assigned categories)</option>
          <option value="admin">Admin (full access)</option>
          <option value="user">User (can browse, cannot upload)</option>
        </select>
      </div>
      <button type="submit" disabled={creating} aria-busy={creating} class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700">
        {creating ? 'Creating…' : 'Create User'}
      </button>
    </form>
  </div>
</div>

<style>
  .row-action-btn {
    padding: 0.2rem 0.5rem;
    font-size: 0.75rem;
    border: 1px solid;
    border-radius: 0.25rem;
    background: white;
    cursor: pointer;
    white-space: nowrap;
    margin-right: 0.25rem;
  }

  .row-action-btn:hover {
    background: #f3f4f6;
  }

  .row-action-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .row-action-delete {
    color: #dc2626;
    border-color: #fecaca;
  }

  .row-action-delete:hover:not(:disabled) {
    background: #fef2f2;
  }
</style>
