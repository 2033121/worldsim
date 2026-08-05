<script>
  import { onMount, onDestroy } from 'svelte';
  import { api } from '../lib/api.js';

  const POLL_INTERVAL = 7000;

  let stats = null;
  let history = null;
  let historyWindow = 'hour'; // hour | day
  let loading = true;
  let historyLoading = false;
  let error = '';
  let historyError = '';
  let lastUpdated = null;
  let timer = null;

  async function loadStats() {
    try {
      stats = await api('GET', '/api/llm/stats');
      error = '';
      lastUpdated = new Date();
    } catch (e) {
      error = e.message || '加载失败';
    } finally {
      loading = false;
    }
  }

  async function loadHistory() {
    historyLoading = true;
    historyError = '';
    try {
      history = await api('GET', '/api/llm/stats/history?window=' + historyWindow);
    } catch (e) {
      historyError = e.message || '历史数据加载失败';
    } finally {
      historyLoading = false;
    }
  }

  function switchWindow(windowName) {
    historyWindow = windowName;
    loadHistory();
  }

  function refreshAll() {
    loadStats();
    loadHistory();
  }

  onMount(() => {
    loadStats();
    loadHistory();
    // 页面可见时每 7 秒刷新实时数据（文档可见性由 mounted 生命周期近似处理）
    timer = setInterval(loadStats, POLL_INTERVAL);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  function fmtNum(n) {
    if (n == null || isNaN(Number(n))) return '-';
    return Number(n).toLocaleString('en-US');
  }

  function fmtUsd(n) {
    if (n == null || isNaN(Number(n))) return '-';
    return '$' + Number(Number(n).toFixed(4)).toLocaleString('en-US');
  }

  function fmtPct(n) {
    if (n == null || isNaN(Number(n))) return '-';
    return (Number(n) * 100).toFixed(1) + '%';
  }

  function fmtTime(ts) {
    if (!ts) return '-';
    const d = new Date(ts);
    if (isNaN(d.getTime())) return String(ts);
    if (historyWindow === 'day') {
      return d.getMonth() + 1 + '/' + d.getDate() + ' ' + String(d.getHours()).padStart(2, '0') + ':00';
    }
    return String(d.getHours()).padStart(2, '0') + ':00';
  }

  // —— 环节分布（横向条形图）——
  $: spans = stats?.summary?.spans || [];
  $: maxSpanTokens = spans.length ? Math.max(...spans.map(s => s.total_tokens || 0), 1) : 1;
  $: maxSpanCalls = spans.length ? Math.max(...spans.map(s => s.calls || 0), 1) : 1;

  // —— 历史趋势（柱状图）——
  $: records = history?.records || [];
  $: maxHistory = records.length
    ? Math.max(...records.map(r => (r.prompt || 0) + (r.completion || 0)), 1)
    : 1;

  $: summary = stats?.summary || {};
  $: estimate = stats?.estimate || {};
  $: cacheDesc = stats?.cache || '';
</script>

<div class="space-y-4">
  <!-- 标题栏 -->
  <div class="flex flex-wrap items-center justify-between gap-2">
    <div>
      <h2 class="text-lg font-semibold">Token 统计</h2>
      <p class="text-sm text-base-content/60">
        {#if lastUpdated}
          <span class="inline-flex items-center gap-1">
            <span class="badge badge-success badge-xs gap-1">
              <span class="loading loading-spinner loading-xs"></span>实时
            </span>
            最近更新 {lastUpdated.toLocaleTimeString()}
          </span>
        {:else}
          LLM 调用用量与预估成本
        {/if}
      </p>
    </div>
    <button class="btn btn-ghost btn-xs gap-1" on:click={refreshAll} title="手动刷新全部数据">
      <span>↻</span> 刷新
    </button>
  </div>

  {#if loading}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body flex items-center justify-center gap-2 py-12 text-base-content/50">
        <span class="loading loading-spinner loading-md"></span>加载中…
      </div>
    </div>
  {:else if error && !stats}
    <div class="alert alert-error shadow-sm">
      <span>{error}（请确认服务已启动，三端口 48090/48091/48092 均可访问）</span>
    </div>
  {:else}
    {#if error}
      <div class="alert alert-error shadow-sm py-2">
        <span>实时刷新失败：{error}</span>
      </div>
    {/if}

    <!-- 实时消耗 -->
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body">
        <div class="flex items-center justify-between">
          <h3 class="card-title text-base">实时消耗</h3>
          <span class="badge badge-ghost badge-sm font-mono">{cacheDesc || '—'}</span>
        </div>
        <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mt-2">
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">调用次数</div>
            <div class="stat-value text-2xl font-mono">{fmtNum(summary.total_calls)}</div>
          </div>
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">输入 Token</div>
            <div class="stat-value text-2xl font-mono">{fmtNum(summary.total_prompt_tokens)}</div>
          </div>
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">输出 Token</div>
            <div class="stat-value text-2xl font-mono">{fmtNum(summary.total_completion_tokens)}</div>
          </div>
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">总 Token</div>
            <div class="stat-value text-2xl font-mono text-primary">{fmtNum(summary.total_tokens)}</div>
          </div>
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">缓存命中率</div>
            <div class="stat-value text-2xl font-mono">{fmtPct(summary.cache_hit_rate)}</div>
          </div>
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">失败</div>
            <div class="stat-value text-2xl font-mono {summary.failures ? 'text-warning' : ''}">{fmtNum(summary.failures)}</div>
          </div>
        </div>
        <div class="divider my-1"></div>
        <div class="flex flex-wrap gap-3 text-sm text-base-content/70">
          <span>缓存 Token：<span class="font-mono">{fmtNum(summary.total_cached_tokens)}</span></span>
        </div>
      </div>
    </div>

    <!-- 环节分布（横向条形图） -->
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body">
        <h3 class="card-title text-base">环节分布</h3>
        {#if spans.length === 0}
          <p class="text-sm text-base-content/50 py-4 text-center">暂无数据</p>
        {:else}
          <div class="space-y-3 mt-2">
            {#each spans as s}
              <div>
                <div class="flex items-center justify-between text-sm mb-1">
                  <span class="font-medium truncate">{s.span}</span>
                  <span class="font-mono text-base-content/70">
                    {fmtNum(s.total_tokens)} tok</span>
                </div>
                <!-- 条形图：整条为 token 占比，内层细分输入/输出 -->
                <div class="flex items-center gap-2">
                  <div class="flex-1 h-4 rounded bg-base-100 overflow-hidden flex">
                    <div
                      class="h-full bg-primary/70"
                      style="width: {Math.max(2, (s.total_tokens / maxSpanTokens) * 100)}%"
                      title="总 token {fmtNum(s.total_tokens)}"
                    ></div>
                  </div>
                  <span class="text-xs font-mono text-base-content/60 w-16 text-right">
                    {fmtNum(s.calls)} 次
                  </span>
                </div>
                <div class="flex gap-4 mt-0.5 text-xs text-base-content/50">
                  <span>输入 {fmtNum(s.prompt_tokens)}</span>
                  <span>输出 {fmtNum(s.completion_tokens)}</span>
                  {#if s.cached_tokens}
                    <span>缓存 {fmtNum(s.cached_tokens)}</span>
                  {/if}
                  {#if s.failures}
                    <span class="text-warning">失败 {fmtNum(s.failures)}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <!-- 历史趋势（柱状图） -->
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h3 class="card-title text-base">历史趋势</h3>
          <div class="flex items-center gap-2">
            <div class="join">
              <button
                class="join-item btn btn-xs {historyWindow === 'hour' ? 'btn-primary' : 'btn-ghost'}"
                on:click={() => switchWindow('hour')}
              >按小时</button>
              <button
                class="join-item btn btn-xs {historyWindow === 'day' ? 'btn-primary' : 'btn-ghost'}"
                on:click={() => switchWindow('day')}
              >按天</button>
            </div>
          </div>
        </div>

        {#if historyLoading}
          <div class="flex items-center justify-center gap-2 py-10 text-base-content/50">
            <span class="loading loading-spinner loading-md"></span>加载中…
          </div>
        {:else if historyError}
          <div class="alert alert-error shadow-sm py-2">
            <span>{historyError}</span>
          </div>
        {:else if records.length === 0}
          <p class="text-sm text-base-content/50 py-6 text-center">暂无历史数据</p>
        {:else}
          <div class="flex items-end gap-1 h-40 mt-3 overflow-x-auto pb-1">
            {#each records as r}
              <div class="flex flex-col items-center justify-end flex-1 min-w-[18px] group" title="时间：{fmtTime(r.ts)}  prompt {fmtNum(r.prompt)} / completion {fmtNum(r.completion)} / calls {fmtNum(r.calls)}">
                <div class="w-full max-w-[28px] rounded-t bg-primary/80 group-hover:bg-primary transition-colors"
                  style="height: {Math.max(2, ((r.prompt || 0) + (r.completion || 0)) / maxHistory * 100)}%"></div>
              </div>
            {/each}
          </div>
          <div class="flex justify-between text-xs text-base-content/50 mt-1">
            <span>{records.length ? fmtTime(records[0].ts) : ''}</span>
            <span>每格 {fmtNum(Math.round(maxHistory))} tok 峰值</span>
            <span>{records.length ? fmtTime(records[records.length - 1].ts) : ''}</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- 预估分析 -->
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body">
        <h3 class="card-title text-base">预估分析</h3>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-2">
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">预估 Token</div>
            <div class="stat-value text-2xl font-mono">{fmtNum(estimate.tokens)}</div>
          </div>
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">预估成本 (USD)</div>
            <div class="stat-value text-2xl font-mono text-accent">{fmtUsd(estimate.usd)}</div>
          </div>
          <div class="stat bg-base-100 rounded-lg py-3">
            <div class="stat-title text-xs">费率 ($/1K tok)</div>
            <div class="stat-value text-2xl font-mono">{fmtUsd(estimate.rate)}</div>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>