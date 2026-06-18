<script lang="ts">
  import { onMount } from 'svelte';
  import { isAuthenticated, navigate } from '../stores/auth';
  import { listCategories, createCategory, deleteCategory } from '../lib/api';

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
  let error: string | null = null;
  let loading = true;

  onMount(async () => {
    if (!$isAuthenticated) {
      navigate('/login');
      return;
    }
    await loadCategories();
  });

  async function loadCategories() {
    error = null;
    loading = true;
    try {
      const data = await listCategories();
      categories = data.categories;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load categories.';
    } finally {
      loading = false;
    }
  }

  async function handleCreate() {
    error = null;
    if (!formName.trim()) {
      error = 'Category name is required.';
      return;
    }
    try {
      await createCategory(formName.trim(), formDescription.trim());
      formName = '';
      formDescription = '';
      await loadCategories();
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to create category.';
    }
  }

  async function handleDelete(id: string) {
    error = null;
    if (!confirm('Are you sure you want to delete this category? Videos in this category will remain.')) {
      return;
    }
    try {
      await deleteCategory(id);
      categories = categories.filter(c => c.id !== id);
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to delete category.';
    }
  }
</script>

<h1>Category Management</h1>

{#if error}
  <article class="error-box">{error}</article>
{/if}

<article>
  <h2>Create Category</h2>
  <form on:submit|preventDefault={handleCreate}>
    <label for="name">
      Name
      <input type="text" id="name" name="name" bind:value={formName} required />
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
    <table role="grid">
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
              <button class="outline secondary" type="button" on:click={() => handleDelete(cat.id)}>
                Delete
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </figure>
{/if}
