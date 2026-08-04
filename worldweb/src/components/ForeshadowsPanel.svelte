<script>
  import { j, escapeHtml } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let foreshadows = '';

  async function load() {
    const d = await j('/api/world/foreshadows');
    foreshadows = d.foreshadows || '';
  }

  $: if ($refreshTick) load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🔗</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">未回收伏笔 <span class="text-xs text-base-content/40 font-normal">防忘坑</span></h2>
  </div>
  {#if !foreshadows}
    <div class="text-sm text-base-content/40">暂无未回收伏笔</div>
  {:else}
    <div class="whitespace-pre-wrap text-sm leading-relaxed text-warning/90 max-h-[40vh] overflow-y-auto p-3 rounded-lg bg-warning/5 border border-warning/20">
      {escapeHtml(foreshadows)}
    </div>
  {/if}
</div>