<script>
  import { j, escapeHtml } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let memories = [];

  async function load() {
    const d = await j('/api/world/memories');
    if (d.memories !== undefined) memories = d.memories;
  }

  $: if ($refreshTick) load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🧠</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">角色记忆</h2>
  </div>
  {#if !memories.length}
    <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">角色记忆将在模拟中沉淀…</div>
  {:else}
    <div class="max-h-[68vh] overflow-y-auto space-y-4 pr-1">
      {#each memories as a (a.actor)}
        <div>
          <div class="text-sm font-semibold text-primary mb-1">🧠 {escapeHtml(a.actor)}</div>
          {#each a.memories as m}
            <div class="flex gap-2 py-1 border-b border-dashed border-base-content/10 text-sm leading-relaxed">
              <span class="shrink-0 text-[10px] text-warning font-bold mt-0.5">{(m.importance * 100).toFixed(0)}%</span>
              <span class="text-base-content/80">{escapeHtml(m.content)}</span>
            </div>
          {/each}
        </div>
      {/each}
    </div>
  {/if}
</div>