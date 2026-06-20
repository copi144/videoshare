<script lang="ts">
  import { onMount } from 'svelte';
  import { listCategories, createCategory, deleteCategory, listUsers, getUploaders, assignUploaders } from '../lib/api';

  export let onError: ((msg: string) => void) | undefined = undefined;

  interface Category {
    name: string;
    display_name: string;
    description: string;
    created_by: string;
    created_at: string;
  }

  interface User {
    name: string;
    display_name: string;
  }

  let categories: Category[] = [];
  let formName = '';
  let formDisplayName = '';
  let formDescription = '';
  let loading = true;

  // Pagination
  let limit = 50;
  let offset = 0;
  let total = 0;

  // Member management modal
  let showMemberModal = false;
  let memberCategoryName = '';
  let allUsers: User[] = [];
  let members: Map<string, boolean> = new Map(); // name → can_upload
  let memberLoading = false;
  let memberSaving = false;
  let memberError: string | null = null;

  onMount(async () => {
    await loadCategories();
  });

  async function loadCategories() {
    loading = true;
    try {
      const data = await listCategories({ limit, offset });
      categories = data.categories;
      total = data.total;
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to load categories.';
      onError?.(msg);
    } finally {
      loading = false;
    }
  }

  async function handleCreate() {
    if (!formName.trim()) {
      onError?.('Category name is required.');
      return;
    }
    try {
      await createCategory(formName.trim(), formDisplayName.trim(), formDescription.trim());
      formName = '';
      formDisplayName = '';
      formDescription = '';
      await loadCategories();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create category.';
      onError?.(msg);
    }
  }

  async function handleDelete(name: string) {
    if (!confirm('Are you sure you want to delete this category? Videos in this category will remain.')) {
      return;
    }
    try {
      await deleteCategory(name);
      await loadCategories();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to delete category.';
      onError?.(msg);
    }
  }

  async function openMembers(name: string) {
    memberCategoryName = name;
    showMemberModal = true;
    memberLoading = true;
    memberError = null;
    members = new Map();
    try {
      // Load all users and current members in parallel
      const [userData, memberData] = await Promise.all([
        listUsers(),
        getUploaders(name),
      ]);
      allUsers = userData.users.map(u => ({ name: u.name, display_name: u.display_name }));
      // Build members map from current assignments
      const m = new Map<string, boolean>();
      for (const mem of memberData.members) {
        m.set(mem.name, mem.can_upload);
      }
      members = m;
    } catch (e: unknown) {
      memberError = e instanceof Error ? e.message : 'Failed to load data.';
    } finally {
      memberLoading = false;
    }
  }

  function closeMembers() {
    showMemberModal = false;
    memberCategoryName = '';
    members = new Map();
    allUsers = [];
    memberError = null;
  }

  function toggleMember(name: string) {
    const next = new Map(members);
    if (next.has(name)) {
      next.delete(name);
    } else {
      next.set(name, false);
    }
    members = next;
  }

  function toggleCanUpload(name: string) {
    const next = new Map(members);
    next.set(name, !next.get(name));
    members = next;
  }

  async function saveMembers() {
    memberSaving = true;
    memberError = null;
    try {
      const memberList = Array.from(members.entries()).map(([name, can_upload]) => ({
        name,
        can_upload,
      }));
      await assignUploaders(memberCategoryName, memberList);
      closeMembers();
    } catch (e: unknown) {
      memberError = e instanceof Error ? e.message : 'Failed to save members.';
    } finally {
      memberSaving = false;
    }
  }
</script>

<div class="space-y-4">
  <!-- Table -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Categories</h2>
    {#if loading}
      <p class="text-gray-500 text-sm">Loading categories…</p>
    {:else if categories.length === 0}
      <p class="text-gray-500 text-sm">No categories yet.</p>
    {:else}
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200">
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Name</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Display Name</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Description</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Created</th>
            <th class="py-2 text-xs font-medium text-gray-500 uppercase">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each categories as cat}
            <tr class="border-b border-gray-100">
              <td class="py-2 pr-4">{cat.name}</td>
              <td class="py-2 pr-4 text-gray-500">{cat.display_name || '—'}</td>
              <td class="py-2 pr-4 text-gray-500">{cat.description || '—'}</td>
              <td class="py-2 pr-4 text-gray-500">{new Date(cat.created_at).toLocaleDateString()}</td>
              <td class="py-2">
                <div class="flex gap-1">
                  <button class="row-action-btn" type="button" on:click={() => openMembers(cat.name)}>Members</button>
                  <button class="row-action-btn row-action-delete" type="button" on:click={() => handleDelete(cat.name)}>Delete</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
      <div class="mt-3 flex items-center gap-2 text-sm">
        <span class="text-gray-500">{offset + 1}–{offset + categories.length} of {total}</span>
        <button type="button" class="row-action-btn" on:click={() => { if (offset > 0) { offset = Math.max(0, offset - limit); loadCategories(); } }} disabled={offset === 0}>Prev</button>
        <button type="button" class="row-action-btn" on:click={() => { offset += limit; loadCategories(); }} disabled={offset + categories.length >= total}>Next</button>
      </div>
    {/if}
  </div>

  <!-- Create form -->
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Create Category</h2>
    <form on:submit|preventDefault={handleCreate} class="space-y-3">
      <div>
        <label for="name" class="block text-sm font-medium text-gray-700 mb-1">Name</label>
        <input type="text" id="name" name="name" bind:value={formName} required pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" class="w-full" />
      </div>
      <div>
        <label for="display_name" class="block text-sm font-medium text-gray-700 mb-1">Display Name</label>
        <input type="text" id="display_name" bind:value={formDisplayName} class="w-full" />
      </div>
      <div>
        <label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea id="description" name="description" bind:value={formDescription} class="w-full"></textarea>
      </div>
      <button type="submit" class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700">Create</button>
    </form>
  </div>
</div>

<!-- Members modal -->
{#if showMemberModal}
  <div class="fixed inset-0 bg-black/30 z-40" on:click={closeMembers}></div>
  <div class="fixed inset-0 flex items-center justify-center z-50">
    <div class="rounded-lg border border-gray-200 bg-white p-6 max-w-lg w-full shadow-xl mx-4">
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-base font-semibold text-gray-900">Members: {memberCategoryName}</h3>
        <button class="text-gray-400 hover:text-gray-600 text-xl leading-none" on:click={closeMembers}>&times;</button>
      </div>

      {#if memberError}
        <div class="rounded-md bg-red-50 border border-red-200 px-3 py-2 text-sm text-red-700 mb-3">{memberError}</div>
      {/if}

      {#if memberLoading}
        <p class="text-sm text-gray-500">Loading members…</p>
      {:else}
        <div class="max-h-64 overflow-y-auto mb-4 space-y-1">
          {#each allUsers as user}
            <div class="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-gray-50">
              <label class="flex items-center gap-2 flex-1 cursor-pointer">
                <input
                  type="checkbox"
                  checked={members.has(user.name)}
                  on:change={() => toggleMember(user.name)}
                />
                <span class="text-sm">{user.display_name || user.name}</span>
              </label>
              {#if members.has(user.name)}
                <label class="flex items-center gap-1 text-xs text-gray-500 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={members.get(user.name)}
                    on:change={() => toggleCanUpload(user.name)}
                  />
                  Upload
                </label>
              {/if}
            </div>
          {/each}
        </div>

        <div class="flex justify-end gap-2 border-t border-gray-100 pt-3">
          <button
            class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50"
            type="button"
            on:click={closeMembers}
          >
            Cancel
          </button>
          <button
            class="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
            type="button"
            disabled={memberSaving}
            on:click={saveMembers}
          >
            {memberSaving ? 'Saving…' : 'Save'}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

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
  .row-action-delete {
    color: #dc2626;
    border-color: #fecaca;
  }
  .row-action-delete:hover {
    background: #fef2f2;
  }
</style>
