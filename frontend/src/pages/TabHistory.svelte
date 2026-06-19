<script lang="ts">
  import { getWatchHistory, clearWatchHistory, removeWatchEntry, getSearchHistory, clearSearchHistory } from '../stores/history';

  let watchHistory = getWatchHistory();
  let searchHistory = getSearchHistory();

  function refresh() {
    watchHistory = getWatchHistory();
    searchHistory = getSearchHistory();
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
</script>

<h2>Watch History</h2>

{#if watchHistory.length === 0}
  <p>No watch history yet. Watch some videos to see them here.</p>
{:else}
  <div class="history-actions">
    <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-500 bg-white hover:bg-gray-100" on:click={() => { clearWatchHistory(); refresh(); }}>Clear All</button>
  </div>
  <table class="w-full text-left divide-y divide-gray-200">
    <thead>
      <tr>
        <th>Title</th>
        <th>Watched</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each watchHistory as entry}
        <tr>
          <td><a href="/s/{entry.id}">{entry.title}</a></td>
          <td>{formatRelativeTime(entry.watchedAt)}</td>
          <td>
            <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-500 bg-white hover:bg-gray-100" on:click={() => { removeWatchEntry(entry.id); refresh(); }}>Remove</button>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}

<h2>Search History</h2>

{#if searchHistory.length === 0}
  <p>No search history yet.</p>
{:else}
  <div class="history-actions">
    <button class="inline-flex items-center px-3 py-1.5 border border-gray-300 rounded-md text-sm text-gray-500 bg-white hover:bg-gray-100" on:click={() => { clearSearchHistory(); refresh(); }}>Clear All</button>
  </div>
  <table class="w-full text-left divide-y divide-gray-200">
    <thead>
      <tr>
        <th>Query</th>
        <th>Searched</th>
      </tr>
    </thead>
    <tbody>
      {#each searchHistory as entry}
        <tr>
          <td>{entry.query}</td>
          <td>{formatRelativeTime(entry.searchedAt)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
