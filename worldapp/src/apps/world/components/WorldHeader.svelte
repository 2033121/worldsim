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

<div class="flex items-center gap-2 px-3 py-1.5 bg-base-200/70 border-b border-base-content/10 shrink-0 flex-wrap">
  <select
    class="select select-sm select-bordered max-w-[180px]"
    value={$currentWorld}
    on:change={(e) => selectWorld(e.target.value)}
    title="选择世界"
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
    class="btn btn-ghost btn-xs btn-circle"
    title="新建世界"
    on:click={() => window.dispatchEvent(new CustomEvent('ws:create-world'))}
  >＋</button>

  {#if $worldState}
    <span class="badge badge-sm badge-ghost">📅 Day <b>{$worldState.day}</b></span>
    <span class="badge badge-sm badge-ghost">🌦 <b>{$worldState.weather || '-'}</b></span>
    <span class="badge badge-sm badge-warning">⚡ {($worldState.world_level?.tension || 0).toFixed(2)}</span>
    <span class="badge badge-sm badge-info">#<b>{$worldState.revision}</b></span>
  {/if}

  <div class="flex-1"></div>
  {#if searchEnabled}
    <span class="badge badge-sm badge-success gap-1" title="联网搜索已启用">🔎 联网搜索</span>
  {/if}
  <button class="btn btn-ghost btn-xs gap-1" on:click={refreshAll}>↻ 刷新</button>
</div>