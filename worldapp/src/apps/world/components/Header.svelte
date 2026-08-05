<script>
  import { onMount } from 'svelte';
  import { j } from '../lib/api.js';
  import { worlds, currentWorld, refreshAll, toast, worldState } from '../lib/stores.js';

  let searchEnabled = false;

  onMount(async () => {
    const d = await j('/api/system/status');
    if (d && typeof d.search_enabled === 'boolean') searchEnabled = d.search_enabled;
  });

  async function selectWorld(name) {
    if (!name) return;
    const d = await j('/api/worlds/select', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ world: name })
    });
    if (d.ok) {
      toast('已切换世界：' + name, 'success');
      refreshAll();
    } else {
      toast(d.error || '切换失败', 'error');
    }
  }
</script>

<header class="navbar bg-base-200/90 backdrop-blur border-b border-base-content/10 px-5 min-h-[52px] shrink-0 gap-3">
  <div class="flex items-center gap-2">
    <span class="text-2xl leading-none cloud-icon">☁</span>
    <span class="text-xl font-bold tracking-widest text-primary">WorldSim</span>
    <span class="text-xs text-base-content/40 mt-1">多世界模拟 · AI 世界演绎</span>
  </div>

  <div class="flex items-center gap-1.5">
    <select
      class="select select-sm select-bordered max-w-[170px]"
      value={$currentWorld}
      on:change={(e) => selectWorld(e.target.value)}
    >
      {#if !$worlds.length}
        <option value="">（无世界）</option>
      {/if}
      {#each $worlds as w (w.name)}
        <option value={w.name} selected={w.name === $currentWorld}>
          {w.active ? '● ' : ''}{w.name}{w.day > 0 ? ' · D' + w.day : ''}
        </option>
      {/each}
    </select>
    <button
      class="btn btn-ghost btn-sm btn-circle"
      title="新建世界"
      on:click={() => window.dispatchEvent(new CustomEvent('ws:create-world'))}
    >＋</button>
  </div>

  <div class="flex items-center gap-1.5 flex-wrap">
    {#if $worldState}
      <span class="badge badge-sm badge-ghost">📅 Day <b>{$worldState.day}</b></span>
      <span class="badge badge-sm badge-ghost">🌦 <b>{$worldState.weather || '-'}</b></span>
      <span class="badge badge-sm badge-warning">⚡ 张力 <b>{($worldState.world_level?.tension || 0).toFixed(2)}</b></span>
      <span class="badge badge-sm badge-info">#<b>{$worldState.revision}</b></span>
    {/if}
  </div>

  <div class="flex-1"></div>
  {#if searchEnabled}
    <span class="badge badge-sm badge-success gap-1" title="已配置联网搜索（SearXNG），Agent 可调用 web_search">🔎 联网搜索</span>
  {/if}
  <button class="btn btn-ghost btn-sm gap-1" on:click={refreshAll}>↻ 刷新</button>
</header>