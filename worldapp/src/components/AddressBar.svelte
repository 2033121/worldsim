<script>
  import { activeTab, navigateTo } from '../lib/browser.js';

  let input = '';
  let editing = false;
  let box;

  $: if (!editing && $activeTab) input = $activeTab.address;

  function commit() {
    const addr = input.trim();
    if (addr) navigateTo(addr);
    editing = false;
    box?.blur();
  }
</script>

<div class="flex items-center gap-1 flex-1 min-w-0">
  <div class="flex-1 min-w-0">
    {#if editing}
      <input
        bind:this={box}
        bind:value={input}
        on:keydown={(e) => { if (e.key === 'Enter') commit(); if (e.key === 'Escape') { editing = false; } }}
        on:blur={() => { editing = false; }}
        class="input input-sm input-bordered w-full text-sm"
        placeholder="输入地址（home / world / novel）回车跳转"
        autofocus
      />
    {:else}
      <button
        class="w-full text-left text-sm px-3 py-1.5 rounded-lg bg-base-200/60 hover:bg-base-300/60 flex items-center gap-2"
        on:click={() => { editing = true; }}
        title="点击编辑地址"
      >
        <span class="opacity-60">🔍</span>
        <span class="truncate text-base-content/80">{input}</span>
      </button>
    {/if}
  </div>
</div>