<script>
  import { j, escapeHtml } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let stats = null;

  async function load() {
    const d = await j('/api/world/token_stats');
    if (!d || d.total_calls === undefined) { stats = null; return; }
    stats = d;
  }

  $: if ($refreshTick) load();

  const rows = (d) => [
    ['总调用次数', d.total_calls || 0],
    ['缓存命中', d.cache_hit_rate || '0%'],
    ['输入 tokens', d.total_prompt_tokens || 0],
    ['输出 tokens', d.total_completion_tokens || 0],
    ['合计 tokens', d.total_tokens || 0]
  ];
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">📊</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">Token 统计</h2>
  </div>
  {#if !stats}
    <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">暂无统计（跑几天模拟后生成）</div>
  {:else}
    <div class="space-y-1.5">
      {#each rows(stats) as r}
        <div class="flex justify-between py-1 border-b border-dashed border-base-content/10 text-sm">
          <span class="text-base-content/50">{r[0]}</span>
          <span class="font-semibold">{r[1]}</span>
        </div>
      {/each}
      {#if stats.spans?.length}
        <div class="text-xs text-base-content/50 mt-3">最近调用：</div>
        <div class="space-y-1">
          {#each stats.spans.slice(-8).reverse() as s}
            <div class="flex justify-between py-1 text-xs">
              <span class="text-base-content/60">{escapeHtml(s.name || s.agent || '')}</span>
              <span class="text-base-content/40">{escapeHtml(s.model || '')} · {s.tokens || 0}t</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>