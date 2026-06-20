<script lang="ts">
  import { onMount } from 'svelte';
  import { listCategories, createCategory, deleteCategory } from '../lib/api';

  export let onError: ((msg: string) => void) | undefined = undefined;

  interface Category {
    name: string;
    display_name: string;
    description: string;
    created_by: string;
    created_at: string;
  }

  let categories: Category[] = [];
  let formName = '';
  let formDisplayName = '';
  let formDescription = '';
  let loading = true;

  // Pagination (local state only)
  let limit = 50;
  let offset = 0;
  let total = 0;

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
                <button class="row-action-btn row-action-delete" type="button" on:click={() => handleDelete(cat.name)}>Delete</button>
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
