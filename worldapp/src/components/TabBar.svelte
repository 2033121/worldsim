<script>
  import { tabs, activeTabId, switchTab, closeTab, openTab } from '../lib/browser.js';
</script>

<div class="flex items-center gap-1 px-2 pt-1 overflow-x-auto flex-1 min-w-0">
  {#each $tabs as t (t.id)}
    <div
      class="tab-item group {activeTabId === t.id ? 'active' : ''}"
      on:click={() => switchTab(t.id)}
      on:contextmenu|preventDefault={() => closeTab(t.id)}
      title="{t.title} （右键关闭）"
    >
      <span class="text-sm leading-none">{t.icon}</span>
      <span class="truncate">{t.title}</span>
      <button
        class="opacity-40 hover:opacity-100 hover:bg-error/20 rounded-full leading-none w-4 h-4 flex items-center justify-center text-xs"
        on:click|stopPropagation={() => closeTab(t.id)}
        title="关闭标签"
      >×</button>
    </div>
  {/each}
  <button
    class="btn btn-ghost btn-xs ml-1 shrink-0"
    on:click={() => openTab('home')}
    title="新建标签 (Ctrl+T)"
  >+</button>
</div>