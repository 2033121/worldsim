<script>
  import { j } from '../lib/api.js';
  import { refreshTick, toast } from '../lib/stores.js';

  let decisions = [];

  async function load() {
    const d = await j('/api/world/decisions');
    decisions = d.decisions || [];
  }

  $: if ($refreshTick) load();

  async function resolve(id, choice) {
    const d = await j('/api/world/decisions/' + id, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ choice })
    });
    if (d.ok) { toast('已改选：' + choice + ' · 写手将按此方向推进', 'success'); load(); }
    else toast('失败：' + (d.error || ''), 'error');
  }

  const statusInfo = (dc) => {
    if (dc.status === 'decided') {
      return dc.user_choice ? { cls: 'badge-warning', txt: '你已改选' } : { cls: 'badge-success', txt: '已定' };
    }
    return { cls: 'badge-info', txt: '待你翻案' };
  };
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🎯</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">决策岔口</h2>
  </div>
  <p class="text-xs text-base-content/50 mb-3">AI 已代决的岔口可随时翻案：点选项即可改选，写手将按你的方向写。</p>

  {#if !decisions.length}
    <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">暂无岔口决策<br>（事件带多个方向时自动入队）</div>
  {:else}
    <div class="max-h-[68vh] overflow-y-auto space-y-3 pr-1">
      {#each decisions as dc (dc.id)}
        {@const info = statusInfo(dc)}
        <div class="rounded-xl bg-base-200/50 border border-base-content/10 p-4">
          <div class="flex items-center gap-2 flex-wrap mb-2">
            <span class="text-xs font-bold text-primary">D{dc.day}</span>
            <span class="badge badge-ghost badge-xs">{dc.type || ''}</span>
            <span class="flex-1 font-semibold text-sm">{dc.title || ''}</span>
            <span class="badge badge-xs {info.cls}">{info.txt}</span>
          </div>
          <div class="text-sm text-base-content/70 bg-base-200/60 border-l-2 border-base-content/20 pl-3 py-2 mb-2 rounded">
            {dc.context || ''}
          </div>
          {#if dc.ai_choice}
            <div class="text-xs text-base-content/60 bg-primary/5 border border-primary/20 rounded-lg p-2 mb-2">
              <b class="text-primary">AI 代决：→ {dc.ai_choice}</b>
              <div class="mt-0.5 text-base-content/50">{dc.ai_reason || ''}</div>
            </div>
          {/if}
          <div class="space-y-1.5">
            {#each dc.options || [] as o}
              {@const chosen = dc.user_choice === o.id || (!dc.user_choice && dc.ai_choice === o.id)}
              <button
                class="opt-btn w-full text-left flex items-start gap-2.5 p-3 rounded-lg border border-base-content/15 bg-base-200/50 text-sm {chosen ? 'chosen' : ''}"
                on:click={() => resolve(dc.id, o.id)}
              >
                <span class="shrink-0 text-[11px] font-bold w-5 h-5 rounded-md bg-primary/15 text-primary flex items-center justify-center">{o.id}</span>
                <span class="flex-1 leading-relaxed">
                  {o.desc}
                  {#if o.reason}<span class="block text-xs text-base-content/50 mt-0.5">→ {o.reason}</span>{/if}
                </span>
              </button>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>