<script>
  import { j } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let events = [];

  async function load() {
    const d = await j('/api/world/today');
    if (!d || !d.day) return;
    const last = d.results ? d.results[d.results.length - 1] : d;
    events = last.events || [];
  }

  $: if ($refreshTick) load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🎲</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">今日事件</h2>
  </div>
  {#if !events.length}
    <div class="text-sm text-base-content/40">无</div>
  {:else}
    <div class="space-y-2">
      {#each events as e}
        <div class="rounded-xl bg-base-200/50 border border-base-content/10 p-4 hover:border-primary/40 transition-all">
          <div class="flex items-center gap-2">
            <span class="font-semibold text-sm text-primary">{e.title}</span>
            <span class="text-xs text-warning shrink-0">sev {(e.severity ?? 0).toFixed(2)}</span>
          </div>
          <div class="text-xs text-base-content/70 leading-relaxed mt-1">{e.frame}</div>
        </div>
      {/each}
    </div>
  {/if}
</div>