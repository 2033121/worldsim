<script>
  import { commandOpen, commandItems, navigateTo, openTab } from '../lib/browser.js';
  import { toggleTheme } from '../lib/theme.js';

  let query = '';
  let box;
  let selected = 0;

  $: list = query
    ? commandItems.filter(i => (i.title + i.hint).toLowerCase().includes(query.toLowerCase()))
    : commandItems;

  $: if (selected >= list.length) selected = 0;

  function go(item) {
    commandOpen.set(false);
    navigateTo(item.address);
  }

  function goNewTab(item) {
    commandOpen.set(false);
    openTab(item.address);
  }

  function close() { commandOpen.set(false); }

  function onKey(e) {
    if (e.key === 'Escape') close();
    else if (e.key === 'ArrowDown') { e.preventDefault(); selected = (selected + 1) % list.length; }
    else if (e.key === 'ArrowUp') { e.preventDefault(); selected = (selected - 1 + list.length) % list.length; }
    else if (e.key === 'Enter') { if (list[selected]) go(list[selected]); }
  }
</script>

{#if $commandOpen}
  <div class="fixed inset-0 z-50 bg-black/40 flex items-start justify-center pt-24 px-4" on:click={close}>
    <div class="palette-enter w-full max-w-lg bg-base-100 rounded-2xl shadow-2xl border border-base-300" on:click|stopPropagation>
      <div class="flex items-center gap-2 px-4 py-3 border-b border-base-300">
        <span class="opacity-60">⌘</span>
        <input
          bind:this={box}
          bind:value={query}
          on:keydown={onKey}
          class="input input-sm input-ghost flex-1 text-sm"
          placeholder="搜索页面、命令…（↑↓选择，Enter 打开，Ctrl+Enter 新标签）"
          autofocus
        />
      </div>
      <div class="max-h-80 overflow-y-auto p-2">
        {#each list as item, i (item.address)}
          <button
            class="w-full text-left px-3 py-2 rounded-lg flex items-center gap-3 hover:bg-base-200 {selected === i ? 'bg-base-300' : ''}"
            on:mouseenter={() => selected = i}
            on:click={() => go(item)}
            on:contextmenu|preventDefault={() => goNewTab(item)}
          >
            <span class="text-lg">{item.icon}</span>
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium">{item.title}</div>
              {#if item.hint}<div class="text-xs opacity-50 truncate">{item.hint}</div>{/if}
            </div>
            <span class="text-xs opacity-40">↩</span>
          </button>
        {/each}
        {#if !list.length}
          <div class="text-center text-sm opacity-50 py-6">无匹配结果</div>
        {/if}
      </div>
      <div class="text-xs opacity-50 px-4 py-2 border-t border-base-300 flex gap-4">
        <span>Enter 打开</span>
        <span>Ctrl+Enter 新标签</span>
        <span>右键 新标签</span>
        <button class="ml-auto text-primary" on:click={toggleTheme}>切换主题</button>
      </div>
    </div>
  </div>
{/if}