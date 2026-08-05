<script>
  import { j } from '../lib/api.js';
  import { refreshTick, toast } from '../lib/stores.js';

  let plans = [];
  let exports = [];
  let generating = false;
  let viewing = null;

  async function load() {
    const d = await j('/api/world/novel');
    if (!d.plans) return;
    plans = d.plans || [];
    exports = d.exports || [];
  }

  $: if ($refreshTick) load();

  async function generate() {
    generating = true;
    toast('小说写手开始创作（约1-3分钟/章）…', 'info');
    const d = await j('/api/world/novel/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({})
    });
    generating = false;
    if (d.ok) {
      toast('✅ 已生成章节：' + ((d.written || []).join('、') || '无新章节'), 'success');
      load();
    } else {
      toast('失败：' + (d.error || ''), 'error');
    }
  }

  async function readChapter(num) {
    const d = await j('/api/world/novel/chapter/' + num);
    if (!d.ok) { toast(d.error || '章节不存在', 'error'); return; }
    viewing = { num, title: d.title, content: d.content };
  }
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-3">
    <span class="text-lg cloud-icon">📖</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">小说章节</h2>
    <div class="flex-1"></div>
    {#if viewing}
      <button class="btn btn-ghost btn-xs" on:click={() => viewing = null}>← 返回列表</button>
    {:else}
      <button class="btn btn-primary btn-xs gap-1" on:click={generate} disabled={generating}>
        {#if generating}<span class="loading loading-spinner loading-xs"></span> 写作中…{:else}✨ 生成小说章节{/if}
      </button>
      <button class="btn btn-ghost btn-xs" on:click={load}>↻</button>
    {/if}
  </div>

  {#if viewing}
    <div class="novel-content text-base-content/85 max-h-[68vh] overflow-y-auto p-5 rounded-lg bg-base-200/50 border border-base-content/10">
      <h3 class="text-center text-lg font-semibold mb-4">{viewing.title}</h3>
      {viewing.content}
    </div>
  {:else}
    {#if !plans.length}
      <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">
        还没有章节。<br>先跑几天模拟，再点「生成小说章节」
      </div>
    {:else}
      <div class="max-h-[60vh] overflow-y-auto space-y-1.5">
        {#each plans as p (p.num)}
          <button class="w-full flex items-center gap-3 p-4 rounded-xl bg-base-200/50 border border-base-content/10 hover:border-primary/40 transition-all cursor-pointer" on:click={() => readChapter(p.num)}>
            <span class="text-xs font-bold text-primary w-8 shrink-0">{String(p.num).padStart(2, '0')}</span>
            <span class="flex-1 font-semibold text-sm text-left">{p.title || ('第' + p.num + '章')}</span>
            <span class="text-xs {p.status === 'done' ? 'text-success' : 'text-base-content/40'} shrink-0">
              {p.status === 'done' ? '✅ 已生成' : '⏳ 待生成'}
            </span>
          </button>
        {/each}
      </div>
      {#if exports.length}
        <div class="text-xs text-base-content/50 mt-3">📄 已导出：{exports.map((e) => e.split('/').pop()).join('、')}</div>
      {/if}
    {/if}
  {/if}
</div>