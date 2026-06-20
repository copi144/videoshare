<script lang="ts">
  import { onMount } from 'svelte';
  import { listUsers, createUser, deleteUser, resetTOTP } from '../lib/api';

  export let onError: ((msg: string) => void) | undefined = undefined;

  interface User {
    name: string;
    display_name: string;
    is_admin: boolean;
    created_at: string;
  }

  let users: User[] = [];
  let loading = true;
  let name = '';
  let displayName = '';
  let isAdmin = false;
  let creating = false;

  let totpResult: { name: string; totp_secret: string; totp_uri: string; qr_image: string } | null = null;

  // Name validation pattern
  const namePattern = /^[0-9A-Za-z\-]*$/;
  $: nameValid = name === '' || namePattern.test(name);

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
    if (!name.trim()) {
      onError?.('Name is required.');
      return;
    }
    creating = true;
    try {
      const result = await createUser(name.trim(), isAdmin, displayName.trim());
      if (result.ok) {
        totpResult = {
          name: name.trim(),
          totp_secret: result.totp_secret,
          totp_uri: result.totp_uri,
          qr_image: result.qr_image,
        };
        name = '';
        displayName = '';
        isAdmin = false;
        await loadUsers();
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create user.';
      onError?.(msg);
    } finally {
      creating = false;
    }
  }

  async function handleDelete(userName: string) {
    if (userName === 'admin') {
      onError?.('Cannot delete the root admin user.');
      return;
    }
    if (!confirm(`Delete user "${userName}"? This cannot be undone.`)) return;
    try {
      await deleteUser(userName);
      await loadUsers();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to delete user.';
      onError?.(msg);
    }
  }

  async function handleResetTOTP(userName: string) {
    if (!confirm(`Reset TOTP for "${userName}"? The user will need to reconfigure their authenticator app.`)) return;
    try {
      const result = await resetTOTP(userName);
      if (result.ok) {
        totpResult = {
          name: userName,
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
      <h2 class="text-base font-semibold text-gray-900 mb-1">TOTP Setup: {totpResult.name}</h2>
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
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Name</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Display Name</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Admin</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Created</th>
            <th class="py-2 text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each users as u}
            <tr class="border-b border-gray-100">
              <td class="py-2 pr-4">{u.name}</td>
              <td class="py-2 pr-4 text-gray-500">{u.display_name || '—'}</td>
              <td class="py-2 pr-4">
                {#if u.is_admin}
                  <span class="inline-flex items-center rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">Yes</span>
                {:else}
                  <span class="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700">No</span>
                {/if}
              </td>
              <td class="py-2 pr-4 text-gray-500">{new Date(u.created_at).toLocaleDateString()}</td>
              <td class="py-2">
                <button class="row-action-btn" type="button" on:click={() => handleResetTOTP(u.name)}>Reset TOTP</button>
                <button class="row-action-btn row-action-delete" type="button" on:click={() => handleDelete(u.name)} disabled={u.name === 'admin'}>Delete</button>
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
    <form on:submit|preventDefault={handleCreate}>
      <!-- Name + Display Name inline -->
      <div class="flex gap-3 mb-3">
        <div class="flex-1">
          <label for="name" class="block text-sm font-medium text-gray-700 mb-1">Name</label>
          <input type="text" id="name" bind:value={name} required pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" class="w-full" class:border-red-500={name !== '' && !nameValid} />
        </div>
        <div class="flex-1">
          <label for="display_name" class="block text-sm font-medium text-gray-700 mb-1">Display Name</label>
          <input type="text" id="display_name" bind:value={displayName} class="w-full" />
        </div>
      </div>

      <!-- Admin toggle: two radio-style buttons -->
      <div class="mb-3">
        <label class="block text-sm font-medium text-gray-700 mb-1">Role</label>
        <div class="inline-flex rounded-md shadow-sm">
          <button
            type="button"
            class="px-4 py-2 text-sm font-medium rounded-l-md border {!isAdmin ? 'bg-indigo-600 text-white border-indigo-600 z-10' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'}"
            on:click={() => isAdmin = false}
          >
            User
          </button>
          <button
            type="button"
            class="px-4 py-2 text-sm font-medium rounded-r-md border-t border-b border-r {isAdmin ? 'bg-indigo-600 text-white border-indigo-600 z-10' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'}"
            on:click={() => isAdmin = true}
          >
            Admin
          </button>
        </div>
      </div>

      <!-- Button row: right-aligned -->
      <div class="flex justify-end">
        <button type="submit" disabled={creating} aria-busy={creating} class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
          {creating ? 'Creating…' : 'Create User'}
        </button>
      </div>
    </form>
  </div>
</div>

<style>
  .row-action-btn {
    padding: 0.2rem 0.5rem;
    font-size: 0.75rem;
    border: 1px solid #d1d5db;
    border-radius: 0.25rem;
    background: white;
    color: #374151;
    cursor: pointer;
    white-space: nowrap;
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
