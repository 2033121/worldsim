<script>
  import { bookmarks, navigateTo, removeBookmark } from '../lib/browser.js';
</script>

{#if $bookmarks.length}
  <div class="flex items-center gap-1 px-2 py-0.5 border-b border-base-300/50 bg-base-200/40 overflow-x-auto">
    <span class="text-xs opacity-50 mr-1 shrink-0">书签</span>
    {#each $bookmarks as b (b.id)}
      <button
        class="text-xs px-2 py-0.5 rounded hover:bg-base-300/60 flex items-center gap-1 shrink-0"
        on:click={() => navigateTo(b.address)}
        title="{b.title} （右键删除书签）"
      >
        <span>{b.icon}</span>
        <span class="truncate max-w-[120px]">{b.title}</span>
      </button>
    {/each}
  </div>
{/if}