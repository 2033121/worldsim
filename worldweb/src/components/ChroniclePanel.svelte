<script>
  import { j, escapeHtml } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let entries = [];
  let readiness = null;

  async function load() {
    const ch = await j('/api/world/chronicle');
    if (ch.chronicle) entries = ch.chronicle;
    const rd = await j('/api/world/readiness');
    if (rd && rd.ready !== undefined) readiness = rd;
  }

  $: if ($refreshTick) load();

  const kindClass = (k) => {
    if (k === 'FACT') return 'tag-FACT';
    if (k === 'SAY') return 'tag-SAY';
    if (k === 'SUMMARY') return 'tag-SUMMARY';
    return 'tag-STATE';
  };
</script>

{#if readiness}
  <div class="paper rounded-xl p-4 mb-3">
    <div class="flex items-center gap-2 mb-2">
      <span class="text-lg cloud-icon">🎬</span>
      <h2 class="text-sm font-semibold text-primary tracking-widest">就绪度 <span class="text-xs text-base-content/40 font-normal">素材够不够写小说</span></h2>
    </div>
    {#if readiness.ready}
      <div class="alert alert-success text-sm py-2">🎬 素材就绪 — 可以生成小说了！</div>
    {:else}
      <div class="alert alert-warning text-sm py-2">⏳ 素材积累中… {readiness.reason || ''}</div>
    {/if}
    <div class="grid grid-cols-2 gap-2 mt-2">
      <div class="text-center py-3 rounded-lg bg-base-200/50 border border-base-content/5">
        <div class="text-xl font-bold text-primary">{readiness.arcs_done}/3</div>
        <div class="text-xs text-base-content/50">完成段落</div>
      </div>
      <div class="text-center py-3 rounded-lg bg-base-200/50 border border-base-content/5">
        <div class="text-xl font-bold text-primary">{readiness.drama_entries}/12</div>
        <div class="text-xs text-base-content/50">戏剧素材</div>
      </div>
      <div class="text-center py-3 rounded-lg bg-base-200/50 border border-base-content/5">
        <div class="text-xl font-bold text-primary">{readiness.foreshadows_resolved}/{readiness.foreshadows_total}</div>
        <div class="text-xs text-base-content/50">伏笔回收</div>
      </div>
      <div class="text-center py-3 rounded-lg bg-base-200/50 border border-base-content/5">
        <div class="text-xl font-bold text-warning">{readiness.tension.toFixed(2)}</div>
        <div class="text-xs text-base-content/50">当前张力</div>
      </div>
    </div>
  </div>
{/if}

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">📜</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">编年史</h2>
  </div>
  {#if !entries.length}
    <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">运行模拟后生成…</div>
  {:else}
    <div class="max-h-[68vh] overflow-y-auto space-y-1 pr-1">
      {#each entries.slice().reverse() as e (e.day + e.content)}
        <div class="chronicle-entry pl-3 py-1.5 text-sm leading-relaxed {e.visibility === 'restricted' || e.visibility === 'private' ? 'opacity-45' : ''}">
          <span class="text-[10px] font-bold mr-1 {kindClass(e.kind)}">{e.kind}</span>
          <span class="text-[10px] text-base-content/40">D{e.day}</span>
          <div class="text-base-content/85">{escapeHtml(e.content)}</div>
        </div>
      {/each}
    </div>
  {/if}
</div>