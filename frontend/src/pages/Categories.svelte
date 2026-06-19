<script lang="ts">
  import { onMount } from 'svelte';
  import { listCategories, createCategory, deleteCategory } from '../lib/api';

  export let onError: ((msg: string) => void) | undefined = undefined;

  interface Category {
    id: string;
    name: string;
    description: string;
    created_by: string;
    created_at: string;
  }

  let categories: Category[] = [];
  let formName = '';
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
      await createCategory(formName.trim(), formDescription.trim());
      formName = '';
      formDescription = '';
      await loadCategories();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to create category.';
      onError?.(msg);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Are you sure you want to delete this category? Videos in this category will remain.')) {
      return;
    }
    try {
      await deleteCategory(id);
      await loadCategories();
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : 'Failed to delete category.';
      onError?.(msg);
    }
  }
</script>

<article>
  <h2>Create Category</h2>
  <form on:submit|preventDefault={handleCreate}>
    <label for="name">
      Name
      <input type="text" id="name" name="name" bind:value={formName} required pattern="[0-9A-Za-z\-]+" title="Letters, numbers, and hyphens only" />
    </label>
    <label for="description">
      Description
      <textarea id="description" name="description" bind:value={formDescription}></textarea>
    </label>
    <button type="submit">Create</button>
  </form>
</article>

<h2>Categories</h2>
{#if loading}
  <p aria-busy="true">Loading categories…</p>
{:else if categories.length === 0}
  <p>No categories yet. Create one above.</p>
{:else}
  <figure>
    <table class="w-full text-left divide-y divide-gray-200">
      <thead>
        <tr>
          <th>Name</th>
          <th>Description</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each categories as cat}
          <tr>
            <td>{cat.name}</td>
            <td>{cat.description || '—'}</td>
            <td>{new Date(cat.created_at).toLocaleDateString()}</td>
            <td>
              <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-500 bg-white hover:bg-gray-100" type="button" on:click={() => handleDelete(cat.id)}>
                Delete
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </figure>

  <!-- Pagination -->
  <div style="margin-top: 0.5rem; display: flex; align-items: center; gap: 0.5rem;">
    <span>{categories.length > 0 ? offset + 1 : 0}–{offset + categories.length} of {total}</span>
    <button type="button" on:click={() => { if (offset > 0) { offset = Math.max(0, offset - limit); loadCategories(); } }} disabled={offset === 0}>Prev</button>
    <button type="button" on:click={() => { offset += limit; loadCategories(); }} disabled={offset + categories.length >= total}>Next</button>
  </div>
{/if}
