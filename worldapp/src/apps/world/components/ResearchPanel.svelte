<script>
  import { j } from '../lib/api.js';
  import { toast, refreshAll } from '../lib/stores.js';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let idea = '';
  let busy = false;
  let current = null; // { record_id, proposal }
  let direction = '';
  let activatingCandidateId = null;
  let history = [];
  let searchOn = false;

  async function loadStatus() {
    const d = await j('/api/system/status');
    searchOn = !!d.search_enabled;
  }
  async function loadHistory() {
    const d = await j('/api/research/proposals');
    history = d.proposals || [];
  }

  async function run() {
    if (!idea.trim()) { toast('先输入你的题材想法', 'error'); return; }
    busy = true;
    current = null;
    direction = '';
    try {
      const d = await j('/api/research', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input: idea.trim() })
      });
      if (d.ok) {
        current = { record_id: d.record_id, proposal: d.proposal };
        toast('研究完成，选出心仪的题材吧', 'success');
        loadHistory();
      } else {
        toast(d.error || '研究失败', 'error');
      }
    } finally {
      busy = false;
    }
  }

  async function genDirection(cand) {
    if (!current) return;
    activatingCandidateId = cand.id;
    direction = '';
    try {
      const d = await j(`/api/research/${current.record_id}/direction`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ candidate_id: cand.id })
      });
      if (d.ok) {
        direction = d.direction || '';
        toast('世界书方向已生成', 'success');
      } else {
        toast(d.error || '方向生成失败', 'error');
      }
    } finally {
      activatingCandidateId = null;
    }
  }

  async function saveCard(cand) {
    if (!current) return;
    activatingCandidateId = cand.id;
    try {
      const d = await j(`/api/research/${current.record_id}/save-card`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ candidate_id: cand.id })
      });
      if (d.ok) {
        toast('题材卡片已沉淀：' + (d.card.name || d.card.id), 'success');
      } else {
        toast(d.error || '存卡失败', 'error');
      }
    } finally {
      activatingCandidateId = null;
    }
  }

  function buildWorld(cand) {
    dispatch('build', { candidate: cand, direction });
  }

  // 加载历史方案
  function viewRecord(rec) {
    if (rec.proposal) {
      current = { record_id: rec.id, proposal: rec.proposal };
      direction = rec.direction || '';
    }
  }

  loadStatus();
  loadHistory();

  function scrollToAttach() {
    const el = [...document.querySelectorAll('h2')].find((h) => h.textContent.includes('世界参考资料'));
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    toast('在左侧「📎 世界参考资料」面板上传，支持 txt/md/json/csv/docx', 'info');
  }
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <span class="text-xl cloud-icon">🔬</span>
    <h3 class="text-lg font-semibold text-primary">题材研究 / 主题规划</h3>
    <span class="badge badge-ghost badge-sm">{searchOn ? '联网搜索已启用' : '未启用联网搜索'}</span>
    <div class="flex-1"></div>
    <button class="btn btn-ghost btn-xs" on:click={scrollToAttach}>📎 上传附件</button>
    <button class="btn btn-ghost btn-xs" on:click={loadHistory}>刷新历史</button>
  </div>

  <div class="paper rounded-2xl p-4 space-y-3">
    <label class="label py-0 text-xs text-base-content/60">你的题材想法（一句话即可，可带附件——点右上"📎 上传附件"上传市场报告/竞品/设定稿，支持 txt/md/docx）</label>
    <textarea class="textarea textarea-bordered w-full min-h-[80px]" bind:value={idea}
      placeholder="如：想做一本修仙+职场的文，主角是给神仙发工资的HR…"></textarea>
    <div class="flex gap-2">
      <button class="btn btn-primary btn-sm flex-1" on:click={run} disabled={busy}>
        {busy ? '研究中…' : '🔍 发起热门题材研究'}
      </button>
    </div>
    {#if busy}
      <div class="alert alert-info text-sm">智能体正在联网研究热门题材 + 综合你的附件素材…（多轮搜索，可能需 1-2 分钟）</div>
    {/if}
  </div>

  {#if current}
    <div class="paper rounded-2xl p-4 space-y-3">
      <h4 class="font-semibold text-base-content/80">候选题材对比方案</h4>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        {#each current.proposal.candidates as cand}
          {@const isRec = cand.id === current.proposal.recommended_id || cand.recommend}
          <div class="border rounded-xl p-3 {isRec ? 'border-primary/60 bg-primary/5' : 'border-base-content/10'}">
            <div class="flex items-center gap-2 mb-1">
              <span class="font-semibold">{cand.title}</span>
              {#if isRec}<span class="badge badge-primary badge-sm">⭐ 推荐</span>{/if}
            </div>
            <div class="text-xs text-base-content/70 space-y-1">
              <p><b>定位：</b>{cand.positioning}</p>
              <p><b>卖点：</b>{cand.selling_point}</p>
              <p><b>风险：</b>{cand.risks}</p>
              <p><b>读者：</b>{cand.audience}</p>
              <p><b>适配：</b>{cand.fit}</p>
              {#if cand.reason}<p class="text-primary"><b>推荐理由：</b>{cand.reason}</p>{/if}
            </div>
            <div class="flex gap-2 mt-2 flex-wrap">
              <button class="btn btn-xs btn-outline" on:click={() => genDirection(cand)} disabled={activatingCandidateId === cand.id && !direction}>
                {activatingCandidateId === cand.id ? '生成中…' : '📋 生成世界书方向'}
              </button>
              <button class="btn btn-xs btn-ghost" on:click={() => saveCard(cand)} disabled={activatingCandidateId === cand.id}>
                存为题材卡片
              </button>
              <button class="btn btn-xs btn-primary" on:click={() => buildWorld(cand)}>据此建世界</button>
            </div>
          </div>
        {/each}
      </div>

      {#if direction}
        <div class="mt-3">
          <h4 class="font-semibold mb-2 text-base-content/80">世界书方向（建世界蓝本）</h4>
          <pre class="whitespace-pre-wrap text-xs bg-base-200/60 rounded-lg p-3 max-h-[40vh] overflow-y-auto">{direction}</pre>
        </div>
      {/if}
    </div>
  {/if}

  {#if history.length > 0}
    <div class="paper rounded-2xl p-4 space-y-2">
      <h4 class="font-semibold text-base-content/80">历史研究</h4>
      {#each history as rec}
        <button class="w-full text-left border border-base-content/10 rounded-lg px-3 py-2 hover:bg-base-200/60" on:click={() => viewRecord(rec)}>
          <div class="text-sm font-medium">{rec.input}</div>
          <div class="text-xs text-base-content/50">{rec.created_at} · {rec.proposal?.candidates?.length || 0} 候选</div>
        </button>
      {/each}
    </div>
  {/if}
</div>