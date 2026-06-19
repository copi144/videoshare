<script lang="ts">
  import { marked } from 'marked';

  export let value: string = '';
  export let placeholder: string = '';

  let activeTab: 'edit' | 'preview' = 'edit';

  $: renderedHtml = value ? marked.parse(value) as string : '<p class="text-gray-400 italic">Nothing to preview</p>';
</script>

<div class="md-editor">
  <!-- Mobile tab bar (visible only on small screens) -->
  <div class="md-tabs">
    <button
      class="md-tab"
      class:active={activeTab === 'edit'}
      on:click={() => { activeTab = 'edit'; }}
    >
      Edit
    </button>
    <button
      class="md-tab"
      class:active={activeTab === 'preview'}
      on:click={() => { activeTab = 'preview'; }}
    >
      Preview
    </button>
  </div>

  <div class="md-panes">
    <div class="md-pane md-editor-pane" class:mobile-hidden={activeTab !== 'edit'}>
      <textarea
        {placeholder}
        bind:value
      ></textarea>
    </div>
    <div class="md-pane md-preview-pane" class:mobile-hidden={activeTab !== 'preview'}>
      <div class="md-preview-content">
        {@html renderedHtml}
      </div>
    </div>
  </div>
</div>

<style>
  .md-editor {
    border: 1px solid #d1d5db;
    border-radius: 0.375rem;
    overflow: hidden;
  }

  .md-tabs {
    display: none;
    border-bottom: 1px solid #d1d5db;
    background: #f9fafb;
  }

  .md-tab {
    flex: 1;
    padding: 0.375rem 0.75rem;
    font-size: 0.8rem;
    border: none;
    background: transparent;
    color: #6b7280;
    cursor: pointer;
    border-bottom: 2px solid transparent;
  }

  .md-tab.active {
    color: #6366f1;
    border-bottom-color: #6366f1;
    font-weight: 500;
  }

  .md-panes {
    display: flex;
    min-height: 160px;
  }

  .md-pane {
    flex: 1;
    min-width: 0;
  }

  .md-editor-pane textarea {
    width: 100%;
    min-height: 160px;
    border: none;
    border-radius: 0;
    resize: vertical;
    padding: 0.5rem;
    font-family: monospace;
    font-size: 0.85rem;
    line-height: 1.5;
    outline: none;
    background: white;
  }

  .md-preview-pane {
    border-left: 1px solid #d1d5db;
    overflow-y: auto;
    max-height: 300px;
  }

  .md-preview-content {
    padding: 0.5rem;
    font-size: 0.85rem;
    line-height: 1.6;
  }

  .md-preview-content :global(h1) { font-size: 1.25rem; font-weight: 700; margin: 0.5rem 0; }
  .md-preview-content :global(h2) { font-size: 1.1rem; font-weight: 600; margin: 0.5rem 0; }
  .md-preview-content :global(h3) { font-size: 1rem; font-weight: 600; margin: 0.4rem 0; }
  .md-preview-content :global(p) { margin: 0.3rem 0; }
  .md-preview-content :global(code) { background: #f3f4f6; padding: 0.1rem 0.25rem; border-radius: 0.2rem; font-size: 0.8rem; }
  .md-preview-content :global(pre) { background: #1f2937; color: #e5e7eb; padding: 0.5rem; border-radius: 0.25rem; overflow-x: auto; margin: 0.4rem 0; }
  .md-preview-content :global(pre code) { background: none; padding: 0; color: inherit; }
  .md-preview-content :global(ul), .md-preview-content :global(ol) { padding-left: 1.25rem; margin: 0.3rem 0; }
  .md-preview-content :global(li) { margin: 0.15rem 0; }
  .md-preview-content :global(blockquote) { border-left: 3px solid #d1d5db; padding-left: 0.75rem; color: #6b7280; margin: 0.3rem 0; }
  .md-preview-content :global(a) { color: #6366f1; text-decoration: underline; }
  .md-preview-content :global(strong) { font-weight: 600; }
  .md-preview-content :global(img) { max-width: 100%; border-radius: 0.25rem; }
  .md-preview-content :global(table) { border-collapse: collapse; width: 100%; margin: 0.3rem 0; }
  .md-preview-content :global(th), .md-preview-content :global(td) { border: 1px solid #d1d5db; padding: 0.25rem 0.5rem; font-size: 0.8rem; }
  .md-preview-content :global(th) { background: #f9fafb; font-weight: 600; }
  .md-preview-content :global(hr) { border: none; border-top: 1px solid #e5e7eb; margin: 0.5rem 0; }

  /* Mobile: hide panes based on activeTab, show tab bar */
  @media (max-width: 768px) {
    .md-tabs {
      display: flex;
    }

    .md-panes {
      flex-direction: column;
    }

    .md-preview-pane {
      border-left: none;
      border-top: 1px solid #d1d5db;
    }

    .mobile-hidden {
      display: none;
    }
  }
</style>
