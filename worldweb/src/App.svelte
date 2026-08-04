<script>
  import { onMount } from 'svelte';
  import { j } from './lib/api.js';
  import {
    worlds, currentWorld, worldState, heroName, loopInfo,
    toasts, activeTab, refreshTick, refreshAll, toast
  } from './lib/stores.js';

  import Header from './components/Header.svelte';
  import ControlPanel from './components/ControlPanel.svelte';
  import WorldStatePanel from './components/WorldStatePanel.svelte';
  import GlobalEventsPanel from './components/GlobalEventsPanel.svelte';
  import ChroniclePanel from './components/ChroniclePanel.svelte';
  import DecisionsPanel from './components/DecisionsPanel.svelte';
  import NovelPanel from './components/NovelPanel.svelte';
  import MemoryPanel from './components/MemoryPanel.svelte';
  import EntitiesPanel from './components/EntitiesPanel.svelte';
  import WorldbookPanel from './components/WorldbookPanel.svelte';
  import StatsPanel from './components/StatsPanel.svelte';
  import ThinkingPanel from './components/ThinkingPanel.svelte';
  import DialoguePanel from './components/DialoguePanel.svelte';
  import EventsPanel from './components/EventsPanel.svelte';
  import ForeshadowsPanel from './components/ForeshadowsPanel.svelte';
  import SnapshotPanel from './components/SnapshotPanel.svelte';
  import CreateWorldModal from './components/CreateWorldModal.svelte';
  import PauseModal from './components/PauseModal.svelte';

  let createWorldModal;
  let pauseModal;

  function onCreateWorld() {
    createWorldModal.show();
  }

  async function loadWorlds() {
    const d = await j('/api/worlds');
    if (!d.worlds) return;
    worlds.set(d.worlds);
    const active = d.worlds.find((w) => w.active);
    if (active) currentWorld.set(active.name);
  }

  async function loadState() {
    const st = await j('/api/world/state');
    if (st && st.day !== undefined) {
      worldState.set(st);
    }
  }

  async function loadLoop() {
    const d = await j('/api/world/loop');
    if (!d || d.running === undefined) return;
    loopInfo.set(d);
  }

  onMount(async () => {
    window.addEventListener('ws:create-world', onCreateWorld);
    await Promise.all([loadWorlds(), loadState(), loadLoop()]);
    setInterval(() => {
      if (!document.hidden) {
        loadWorlds();
        loadState();
        loadLoop();
        refreshAll(); // 触发各面板重新加载，实现自动刷新
      }
    }, 8000);
    return () => window.removeEventListener('ws:create-world', onCreateWorld);
  });

  $: if ($activeTab === 'novel' || $activeTab === 'memory' || $activeTab === 'entities') {
    // 切换到这些 tab 时刷新
  }
</script>

<div class="flex flex-col h-screen bg-base-200 text-base-content overflow-hidden">
  <Header />

  <div class="flex flex-1 overflow-hidden">
    <!-- 左栏：操作台 + 世界状态 + 全局事件 + 快照 -->
    <aside class="w-72 shrink-0 overflow-y-auto p-5 space-y-5 border-r border-base-content/10 bg-base-100/60">
      <ControlPanel />
      <SnapshotPanel />
      <WorldStatePanel />
      <GlobalEventsPanel />
    </aside>

    <!-- 中栏：Tab 内容 -->
    <main class="flex-1 min-w-0 overflow-y-auto bg-base-100/40">
      <div class="flex gap-2 p-5 pb-0 flex-wrap">
        {#each [
          ['chronicle', '📜', '编年史'],
          ['decisions', '🎯', '决策'],
          ['novel', '📖', '小说'],
          ['memory', '🧠', '记忆'],
          ['entities', '🧍', '实体'],
          ['worldbook', '📚', '世界书'],
          ['stats', '📊', '统计']
        ] as [id, icon, label]}
          <button
            class="btn btn-md {activeTab === id ? 'btn-primary font-medium' : 'btn-ghost'} gap-1.5"
            on:click={() => activeTab.set(id)}
          >
            <span>{icon}</span>{label}
          </button>
        {/each}
      </div>
      <div class="p-5">
        {#if $activeTab === 'chronicle'}
          <ChroniclePanel />
        {:else if $activeTab === 'decisions'}
          <DecisionsPanel />
        {:else if $activeTab === 'novel'}
          <NovelPanel />
        {:else if $activeTab === 'memory'}
          <MemoryPanel />
        {:else if $activeTab === 'entities'}
          <EntitiesPanel />
        {:else if $activeTab === 'worldbook'}
          <WorldbookPanel />
        {:else if $activeTab === 'stats'}
          <StatsPanel />
        {/if}
      </div>
    </main>

    <!-- 右栏：主角内心 + 今日对话 + 今日事件 + 伏笔 -->
    <aside class="w-80 shrink-0 overflow-y-auto p-5 space-y-5 border-l border-base-content/10 bg-base-100/60">
      <ThinkingPanel />
      <DialoguePanel />
      <EventsPanel />
      <ForeshadowsPanel />
    </aside>
  </div>

  <!-- Toasts -->
  <div class="fixed top-5 right-5 z-50 flex flex-col gap-2">
    {#each $toasts as t (t.id)}
      <div class="alert alert-sm {t.type === 'success' ? 'alert-success' : t.type === 'error' ? 'alert-error' : 'alert-info'} toast-enter shadow-lg max-w-sm">
        <span>{t.msg}</span>
      </div>
    {/each}
  </div>

  <CreateWorldModal bind:this={createWorldModal} />
  <PauseModal bind:this={pauseModal} />
</div>