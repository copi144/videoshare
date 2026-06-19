<script lang="ts">
  export let show = false;
  export let title = 'Confirm';
  export let message = 'Are you sure?';
  export let confirmLabel = 'Confirm';
  export let onConfirm: () => void = () => {};
  export let onCancel: () => void = () => {};

  let input = '';
  $: isValid = input.toLowerCase() === 'yes';

  function handleConfirm() {
    if (isValid) {
      onConfirm();
      input = '';
    }
  }

  function handleCancel() {
    onCancel();
    input = '';
  }
</script>

{#if show}
  <!-- Backdrop -->
  <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center" on:click={handleCancel}>
    <!-- Modal -->
    <div class="bg-white rounded-lg shadow-xl p-6 max-w-md w-full mx-4" on:click|stopPropagation>
      <h3 class="text-lg font-semibold mb-2">{title}</h3>
      <p class="text-gray-600 mb-4">{message}</p>
      <input
        type="text"
        placeholder='Type "yes" to confirm'
        bind:value={input}
        on:keydown={(e) => { if (e.key === 'Enter' && isValid) handleConfirm(); if (e.key === 'Escape') handleCancel(); }}
        class="w-full border border-gray-300 rounded-md px-3 py-2 mb-4 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        autofocus
      />
      <div class="flex justify-end gap-2">
        <button class="px-4 py-2 border border-gray-300 rounded-md text-sm text-gray-700 hover:bg-gray-50" on:click={handleCancel}>
          Cancel
        </button>
        <button
          class="px-4 py-2 rounded-md text-sm text-white disabled:opacity-50"
          class:bg-red-600={!title.includes('Retranscode')}
          class:bg-indigo-600={title.includes('Retranscode')}
          disabled={!isValid}
          on:click={handleConfirm}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}
