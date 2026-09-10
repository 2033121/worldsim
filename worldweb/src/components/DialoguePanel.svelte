<script>
  import { j } from '../lib/api.js';
  import { refreshTick, heroName, lastResult } from '../lib/stores.js';

  let dialogue = [];

  async function load() {
    const d = await j('/api/world/today');
    if (!d || !d.day) return;
    const last = d.results ? d.results[d.results.length - 1] : d;
    dialogue = last.dialogue || [];
  }

  $: if ($refreshTick) load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">💬</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">今日对话</h2>
  </div>
  {#if !dialogue.length}
    <div class="text-sm text-base-content/40">今日无对话（平淡日）</div>
  {:else}
    <div class="space-y-1.5">
      {#each dialogue as t}
        {@const me = t.speaker.includes($heroName) || t.speaker === 'protagonist'}
        <div class="flex gap-2 text-sm leading-relaxed">
          <span class="shrink-0 text-[11px] font-bold px-2 py-0.5 rounded-md self-start {me ? 'bg-success/15 text-success' : 'bg-secondary/15 text-secondary'}">
            {t.speaker.replace('npc_', '')}
          </span>
          <span class="text-base-content/85">{t.speech}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>