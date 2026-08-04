<script>
  import { activeTab, canGoBack, canGoForward, goBack, goForward, commandOpen, isBookmarked, addBookmark, removeBookmark, statusText } from '../lib/browser.js';
  import { ROUTES } from '../lib/routes.js';
  import TabBar from './TabBar.svelte';
  import AddressBar from './AddressBar.svelte';
  import ThemeToggle from './ThemeToggle.svelte';

  $: addr = $activeTab?.address || 'home';
  $: bookmarked = isBookmarked(addr);
  $: statusText.set(($activeTab && ROUTES[addr]?.hint) || '就绪');
</script>

<div class="bg-base-200/80 border-b border-base-300/60 backdrop-blur shrink-0 select-none">
  <!-- 标签栏 -->
  <TabBar />

  <!-- 工具栏：导航按钮 + 地址栏 + 书签 + 主题 -->
  <div class="flex items-center gap-1.5 px-2 pb-1.5">
    <button class="btn btn-ghost btn-sm" disabled={!canGoBack()} on:click={goBack} title="后退 (Alt+←)">←</button>
    <button class="btn btn-ghost btn-sm" disabled={!canGoForward()} on:click={goForward} title="前进 (Alt+→)">→</button>
    <button class="btn btn-ghost btn-sm" on:click={() => commandOpen.set(true)} title="命令面板 (Ctrl+K)">⌘</button>

    <AddressBar />

    <button
      class="btn btn-ghost btn-sm {bookmarked ? 'text-warning' : ''}"
      on:click={() => bookmarked ? removeBookmark(addr) : addBookmark(addr)}
      title="{bookmarked ? '取消书签' : '加入书签'}"
    >{bookmarked ? '★' : '☆'}</button>
    <ThemeToggle />
  </div>
</div>