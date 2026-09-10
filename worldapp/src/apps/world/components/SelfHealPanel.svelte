<script>
  import { j } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let status = null;
  let incidents = [];

  async function load() {
    const s = await j('/api/selfheal/status');
    if (s && s.enabled !== undefined) status = s;
    const inc = await j('/api/selfheal/incidents');
    if (inc && Array.isArray(inc.incidents)) incidents = inc.incidents;
  }

  $: if ($refreshTick) load();
  $: $refreshTick; // 依赖刷新信号

  const sevClass = (sev) => ({
    ok: 'badge-success',
    warn: 'badge-warning',
    error: 'badge-error',
    critical: 'badge-error',
  }[sev] || 'badge-ghost');

  const catLabel = {
    llm_api: 'LLM/API',
    simulation: '模拟',
    process: '服务',
    data: '数据',
  };

  function uptime(sec) {
    if (!sec) return '-';
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    return h > 0 ? `${h}小时${m}分` : `${m}分`;
  }
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-3">
    <span class="text-lg cloud-icon">🛠️</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">运行监测与自愈</h2>
  </div>

  {#if !status}
    <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">
      自愈模块未启用
    </div>
  {:else}
    <!-- 概览 -->
    <div class="grid grid-cols-2 gap-2 mb-3">
      <div class="rounded-lg bg-base-200/60 p-3">
        <div class="text-xs text-base-content/50">运行时长</div>
        <div class="font-semibold">{uptime(status.uptime_sec)}</div>
      </div>
      <div class="rounded-lg bg-base-200/60 p-3">
        <div class="text-xs text-base-content/50">自动修复</div>
        <div class="font-semibold">{status.repairs || 0} 次</div>
      </div>
    </div>

    <!-- LLM 状态 -->
    <div class="flex items-center gap-2 mb-3 p-3 rounded-lg {status.llm_ready ? 'bg-success/10' : 'bg-warning/15'}">
      <span class="badge {status.llm_ready ? 'badge-success' : 'badge-warning'}">
        {status.llm_ready ? 'LLM 就绪' : 'LLM 未就绪'}
      </span>
      <span class="text-xs text-base-content/70 truncate">{status.llm_detail || ''}</span>
    </div>

    <!-- 各监测项 -->
    <div class="space-y-1.5 mb-4">
      {#each status.checks || [] as c}
        <div class="flex items-center justify-between py-1.5 border-b border-dashed border-base-content/10">
          <div class="flex items-center gap-2 min-w-0">
            <span class="badge badge-sm {sevClass(c.severity)}">{c.severity}</span>
            <span class="text-sm">{c.name}</span>
          </div>
          <span class="text-xs text-base-content/50 truncate ml-2">{c.detail}</span>
        </div>
      {/each}
    </div>

    <!-- 检测与修复记录 -->
    <div class="text-xs text-base-content/50 mb-1.5">检测与修复记录（{incidents.length}）：</div>
    {#if incidents.length === 0}
      <div class="text-xs text-base-content/40 text-center py-3 border border-dashed border-base-content/20 rounded-lg">
        暂无异常记录
      </div>
    {:else}
      <div class="space-y-2 max-h-72 overflow-y-auto pr-1">
        {#each incidents as inc}
          <div class="rounded-lg border border-base-content/10 p-2.5">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="badge badge-sm {sevClass(inc.severity)}">{inc.severity}</span>
              <span class="badge badge-sm badge-ghost">{catLabel[inc.category] || inc.category}</span>
              <span class="text-xs text-base-content/40">{inc.time}</span>
              {#if inc.auto_fixed}
                <span class="badge badge-sm badge-success ml-auto">已自动修复</span>
              {:else if inc.status === 'manual'}
                <span class="badge badge-sm badge-warning ml-auto">需人工处理</span>
              {/if}
            </div>
            <div class="text-sm mt-1.5">{inc.detail}</div>
            <div class="text-xs text-base-content/50 mt-1">诊断：{inc.diagnosis}</div>
            <div class="text-xs text-primary mt-0.5">修复：{inc.action}</div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>