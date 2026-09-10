<script>
  import { j, MODELS } from '../lib/api.js';
  import { heroName, currentWorld, refreshAll, toast, lastResult } from '../lib/stores.js';
  import PauseModal from './PauseModal.svelte';

  let pauseModal;

  let hero = '';
  let llmMode = 'real';
  let baseUrl = 'https://tokenrhythm.studio/v1';
  let modelName = 'deepseek-v4-flash';
  let apiKey = '';
  let tierFast = 'deepseek-v4-flash-0731';
  let tierNormal = 'deepseek-v4-pro';
  let tierPremium = 'deepseek-v4-pro';
  let daysNum = 1;
  let simMode = 'auto';
  let loopDays = 1000;
  let loopMode = 'auto';
  let running = false;
  let loopRunning = false;
  let loopProgress = 0;
  let loopDetail = '循环未启动';

  async function initWorld() {
    const name = hero.trim();
    const d = await j('/api/world/init', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ protagonist: name })
    });
    if (d.ok) {
      toast('世界初始化完成 · 主角：' + (name || '（世界书自拟）'), 'success');
      heroName.set(name);
      refreshAll();
    } else {
      toast('失败：' + (d.error || ''), 'error');
    }
  }

  async function setLLM() {
    const body = { mode: llmMode };
    if (llmMode === 'real') {
      body.base_url = baseUrl.trim();
      body.model = modelName.trim();
      body.api_key = apiKey.trim();
      body.model_tiers = { fast: tierFast, normal: tierNormal, premium: tierPremium };
    }
    const d = await j('/api/world/sim/llm', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (d.ok) toast('LLM 模式：' + llmMode + ' · 分层已生效', 'success');
    else toast('失败：' + (d.error || ''), 'error');
  }

  async function runDays() {
    const days = Math.min(parseInt(daysNum) || 1, 30);
    running = true;
    const d = await j('/api/world/sim/day', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ days, mode: simMode })
    });
    running = false;
    if (!d.ok) { toast('失败：' + (d.error || ''), 'error'); return; }
    lastResult.set(d);
    toast('模拟完成：跑至 Day ' + d.day, 'success');
    refreshAll();
    if (d.paused) pauseModal.show(d);
  }

  async function startLoop() {
    const d = await j('/api/world/loop', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'start', days: parseInt(loopDays) || 1000, mode: loopMode })
    });
    if (d.ok) toast('循环启动 → 目标 Day ' + d.target_day, 'success');
    else toast('失败：' + (d.error || ''), 'error');
    fetchLoop();
  }

  async function stopLoop() {
    await j('/api/world/loop', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'stop' })
    });
    toast('已停止循环');
    fetchLoop();
  }

  async function fetchLoop() {
    const d = await j('/api/world/loop');
    if (!d || d.running === undefined) return;
    loopRunning = d.running;
    if (d.running) {
      loopProgress = Math.min(100, Math.round((d.day / (d.target_day || 1)) * 100));
      loopDetail = `运行中 · ${d.world || ''} · Day ${d.day} → 目标 ${d.target_day}${d.ready ? ' · 🎬 已就绪' : ''}`;
    } else {
      if (d.target_day > 0 && d.day > 0) {
        loopDetail = `已停止 · Day ${d.day}${d.ready ? ' · 🎬 素材就绪，可以写小说了！' : ' · 未就绪'}`;
      } else {
        loopDetail = '循环未启动';
      }
    }
  }
</script>

<div class="paper rounded-xl p-4 space-y-3">
  <div class="flex items-center gap-2">
    <span class="text-lg cloud-icon">⚙</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">操作台</h2>
    <div class="flex-1"></div>
    <span class="badge badge-outline badge-xs">{ $currentWorld || '未选择世界' }</span>
  </div>

  <div>
    <label class="label py-0 text-xs text-base-content/60">主角名</label>
    <div class="flex gap-2">
      <input class="input input-sm input-bordered flex-1" placeholder="主角名" bind:value={hero} />
      <button class="btn btn-primary btn-sm" on:click={initWorld}>初始化</button>
    </div>
  </div>

  <div>
    <label class="label py-0 text-xs text-base-content/60">LLM 模式</label>
    <div class="flex gap-2">
      <select class="select select-sm select-bordered flex-1" bind:value={llmMode}>
        <option value="real">real · 真实LLM</option>
        <option value="mock">mock · 本地模拟</option>
        <option value="off">off · 模板</option>
      </select>
      <button class="btn btn-ghost btn-sm" on:click={setLLM}>应用</button>
    </div>
  </div>

  {#if llmMode === 'real'}
  <div class="space-y-1.5">
    <input class="input input-sm input-bordered w-full" placeholder="Base URL" bind:value={baseUrl} />
    <input class="input input-sm input-bordered w-full" placeholder="模型" bind:value={modelName} />
    <input class="input input-sm input-bordered w-full" placeholder="API Key" type="password" bind:value={apiKey} />
    <div class="text-xs text-base-content/50 pt-1">🧠 模型分层（各司其职）</div>
    <div class="flex items-center gap-1.5">
      <span class="text-xs text-base-content/50 w-14 shrink-0">⚡ 高频</span>
      <select class="select select-xs select-bordered flex-1" bind:value={tierFast}>
        {#each MODELS as m}<option value={m}>{m}</option>{/each}
      </select>
    </div>
    <div class="flex items-center gap-1.5">
      <span class="text-xs text-base-content/50 w-14 shrink-0">🔷 推理</span>
      <select class="select select-xs select-bordered flex-1" bind:value={tierNormal}>
        {#each MODELS as m}<option value={m}>{m}</option>{/each}
      </select>
    </div>
    <div class="flex items-center gap-1.5">
      <span class="text-xs text-base-content/50 w-14 shrink-0">💎 产出</span>
      <select class="select select-xs select-bordered flex-1" bind:value={tierPremium}>
        {#each MODELS as m}<option value={m}>{m}</option>{/each}
      </select>
    </div>
    <div class="text-[10px] text-base-content/40">⚡事件/NPC · 🔷主角/GM/反思 · 💎小说/抉择</div>
  </div>
  {/if}

  <div>
    <label class="label py-0 text-xs text-base-content/60">单次模拟</label>
    <div class="flex gap-2">
      <input class="input input-sm input-bordered w-16" type="number" min="1" max="30" bind:value={daysNum} />
      <select class="select select-sm select-bordered flex-1" bind:value={simMode}>
        <option value="auto">⚙️ 自动</option>
        <option value="scene">🎬 Scene</option>
        <option value="summary">📝 Summary</option>
        <option value="skip">⏩ Skip</option>
      </select>
      <button class="btn btn-primary btn-sm" on:click={runDays} disabled={running}>
        {#if running}<span class="loading loading-spinner loading-xs"></span>{:else}▶ 跑{/if}
      </button>
    </div>
  </div>

  <div class="scroll-rule"></div>

  <div>
    <label class="label py-0 text-xs text-base-content/60">🔁 后台持续运行（到就绪自动停）</label>
    <div class="flex gap-2">
      <input class="input input-sm input-bordered w-20" type="number" min="1" bind:value={loopDays} />
      <select class="select select-sm select-bordered flex-1" bind:value={loopMode}>
        <option value="auto">⚙️ 自动</option>
        <option value="scene">🎬 Scene</option>
        <option value="summary">📝 Summary</option>
      </select>
      {#if !loopRunning}
        <button class="btn btn-primary btn-sm" on:click={startLoop}>▶ 开始</button>
      {:else}
        <button class="btn btn-error btn-sm" on:click={stopLoop}>⏹ 停止</button>
      {/if}
    </div>
    <progress class="progress progress-primary w-full h-1.5 mt-2" value={loopProgress} max="100"></progress>
    <div class="text-[11px] text-base-content/50 mt-1">{loopDetail}</div>
  </div>
</div>

<PauseModal bind:this={pauseModal} />