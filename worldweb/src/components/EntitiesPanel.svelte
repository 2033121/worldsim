<script>
  import { worldState } from '../lib/stores.js';
  import { escapeHtml } from '../lib/api.js';
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🧍</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">实体</h2>
  </div>
  {#if !$worldState?.entities || !Object.keys($worldState.entities).length}
    <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">初始化后显示</div>
  {:else}
    <div class="grid gap-2 sm:grid-cols-2 pr-1">
      {#each Object.entries($worldState.entities) as [k, e]}
        <div class="rounded-xl bg-base-200/50 border border-base-content/10 p-4 hover:border-primary/40 transition-all">
          <div class="font-semibold text-sm">{escapeHtml(k)}</div>
          <div class="text-xs text-base-content/50 mt-1 leading-relaxed">
            {escapeHtml(e.job || '')} · {escapeHtml(e.location || '')}{e.health ? ' · 健康' + e.health : ''}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>