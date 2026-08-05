<script>
  import { worldState } from '../lib/stores.js';
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
          <div class="font-semibold text-sm">{k}</div>
          <div class="text-xs text-base-content/50 mt-1 leading-relaxed">
            {e.job || ''} · {e.location || ''}
            <div class="mt-1 flex flex-wrap gap-1">
              {#if e.assets && Object.keys(e.assets).length}
                {#each Object.entries(e.assets) as [aname, aval]}
                  <span class="badge badge-sm badge-outline">{aname} {aval}</span>
                {/each}
              {/if}
              {#if e.money != null && !(e.assets && Object.keys(e.assets).length)}
                <span class="badge badge-sm badge-outline">💰 {e.money}</span>
              {/if}
            </div>
            <div class="mt-1">
              {#if e.body && (Object.keys(e.body.vitals || {}).length || e.body.desc)}
                <div class="flex flex-wrap gap-1">
                  {#each Object.entries(e.body.vitals || {}) as [vname, vval]}
                    <span class="badge badge-sm badge-primary badge-outline">{vname} {vval}</span>
                  {/each}
                </div>
                {#if e.body.desc}
                  <div class="text-xs text-base-content/60 mt-1 italic">"{e.body.desc}"</div>
                {/if}
              {:else if e.health != null}
                <span class="badge badge-sm badge-secondary badge-outline">健康 {e.health}</span>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>