<script lang="ts">
  import { getWatchHistory, clearWatchHistory, removeWatchEntry, getSearchHistory, clearSearchHistory } from '../stores/history';

  let watchHistory = getWatchHistory();
  let searchHistory = getSearchHistory();

  // Search
  let searchQuery = '';
  $: filteredWatchHistory = searchQuery
    ? watchHistory.filter(e => e.title.toLowerCase().includes(searchQuery.toLowerCase()))
    : watchHistory;

  // Pagination
  const watchLimit = 20;
  let watchOffset = 0;
  $: pagedWatchHistory = filteredWatchHistory.slice(watchOffset, watchOffset + watchLimit);
  $: watchTotal = filteredWatchHistory.length;

  function refresh() {
    watchHistory = getWatchHistory();
    searchHistory = getSearchHistory();
    watchOffset = 0;
  }

  function formatRelativeTime(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    return `${Math.floor(days / 30)}mo ago`;
  }

  function handlePrevPage() {
    if (watchOffset > 0) {
      watchOffset = Math.max(0, watchOffset - watchLimit);
    }
  }

  function handleNextPage() {
    if (watchOffset + watchLimit < watchTotal) {
      watchOffset += watchLimit;
    }
  }
</script>

<div class="space-y-4">
  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <!-- Title row with search -->
    <div class="flex items-center mb-3">
      <h2 class="text-base font-semibold text-gray-900 flex-1">Watch History</h2>
      <input
        type="text"
        placeholder="Search title…"
        bind:value={searchQuery}
        class="w-48 text-sm mr-2"
      />
      <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50" on:click={() => { if (confirm('Are you sure you want to clear all history? This cannot be undone.')) { clearWatchHistory(); refresh(); } }}>Clear All</button>
    </div>
    {#if watchHistory.length === 0}
      <p class="text-gray-500 text-sm">No watch history yet. Watch some videos to see them here.</p>
    {:else if filteredWatchHistory.length === 0}
      <p class="text-gray-500 text-sm">No results match your search.</p>
    {:else}
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200">
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Title</th>
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Watched</th>
            <th class="py-2 text-xs font-medium text-gray-500 uppercase"></th>
          </tr>
        </thead>
        <tbody>
          {#each pagedWatchHistory as entry}
            <tr class="border-b border-gray-100">
              <td class="py-2 pr-4"><a href="/#/v/{entry.id}" class="text-indigo-600 hover:text-indigo-800 underline">{entry.title}</a></td>
              <td class="py-2 pr-4 text-gray-500">{formatRelativeTime(entry.watchedAt)}</td>
              <td class="py-2">
                <button class="row-action-btn row-action-delete" type="button" on:click={() => { removeWatchEntry(entry.id); refresh(); }}>Remove</button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
      <!-- Pagination -->
      <div class="mt-3 flex items-center gap-2 text-sm">
        <span class="text-gray-500">{watchOffset + 1}&ndash;{Math.min(watchOffset + watchLimit, watchTotal)} of {watchTotal}</span>
        <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed" type="button" on:click={handlePrevPage} disabled={watchOffset === 0}>Prev</button>
        <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed" type="button" on:click={handleNextPage} disabled={watchOffset + watchLimit >= watchTotal}>Next</button>
      </div>
    {/if}
  </div>

  <div class="rounded-lg border border-gray-200 bg-white p-4">
    <h2 class="text-base font-semibold text-gray-900 mb-3">Search History</h2>
    {#if searchHistory.length === 0}
      <p class="text-gray-500 text-sm">No search history yet.</p>
    {:else}
      <div class="mb-3">
        <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-700 bg-white hover:bg-gray-50" on:click={() => { if (confirm('Are you sure you want to clear all history? This cannot be undone.')) { clearSearchHistory(); refresh(); } }}>Clear All</button>
      </div>
      <table class="w-full text-left text-sm">
        <thead>
          <tr class="border-b border-gray-200">
            <th class="py-2 pr-4 text-xs font-medium text-gray-500 uppercase">Query</th>
            <th class="py-2 text-xs font-medium text-gray-500 uppercase">Searched</th>
          </tr>
        </thead>
        <tbody>
          {#each searchHistory as entry}
            <tr class="border-b border-gray-100">
              <td class="py-2 pr-4">{entry.query}</td>
              <td class="py-2 text-gray-500">{formatRelativeTime(entry.searchedAt)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
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
  .row-action-delete {
    color: #dc2626;
    border-color: #fecaca;
  }
  .row-action-delete:hover {
    background: #fef2f2;
  }
</style>
