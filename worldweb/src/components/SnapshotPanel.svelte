<script>
  import { j } from '../lib/api.js';
  import { refreshAll, toast } from '../lib/stores.js';

  let snapshots = [];
  let reason = '';

  async function load() {
    const d = await j('/api/world/snapshots');
    snapshots = d.snapshots || [];
  }

  async function makeSnapshot() {
    const r = reason.trim() || '手动存档';
    const d = await j('/api/world/snapshot', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason: r })
    });
    if (d.ok) {
      toast('💾 已存档 Day' + d.snapshot.day + '（' + r + '）', 'success');
      reason = '';
      load();
    } else {
      toast('失败：' + (d.error || ''), 'error');
    }
  }

  async function rewindTo(day) {
    if (!confirm(`确定回退到 Day${day}？\n当前之后的一切（编年史/记忆/决策/伏笔）都会被该时间点的状态覆盖，并从那里重新演化出新分支。`)) return;
    const d = await j('/api/world/rewind', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ day })
    });
    if (d.ok) {
      toast('⏪ 已回退到 Day' + d.rewound_to + ' · 可重新开始循环', 'success');
      refreshAll();
    } else {
      toast('回退失败：' + (d.error || ''), 'error');
    }
  }

  load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">⏪</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">时间回退</h2>
  </div>
  <div class="flex gap-2">
    <input class="input input-sm input-bordered flex-1" placeholder="存档说明（可选）" bind:value={reason} />
    <button class="btn btn-primary btn-sm" on:click={makeSnapshot}>💾 存档</button>
  </div>
  <div class="mt-2 max-h-40 overflow-y-auto space-y-1">
    {#if !snapshots.length}
      <div class="text-[11px] text-base-content/40">暂无快照（每30天自动存，或点💾手动存）</div>
    {:else}
      {#each snapshots as s (s.day)}
        <div class="flex items-center gap-2 py-1 border-b border-dashed border-base-content/10 text-xs">
          <span class="text-primary font-bold shrink-0">D{s.day}</span>
          <span class="flex-1 text-base-content/50 truncate">{s.reason || ''}</span>
          <button class="btn btn-ghost btn-xs" on:click={() => rewindTo(s.day)} title="回退到 D{s.day}">⏪ 回退</button>
        </div>
      {/each}
    {/if}
  </div>
</div>