<script>
  // StudioPanel — 美术工坊面板：世界书驱动的像素素材生成（规划 → 生成 → 画廊）。
  // 完整编辑体验在独立页 /studio（终端风自包含页）；本面板提供常用入口与概览。
  import { j } from '../lib/api.js';
  import { refreshTick, toasts, toast } from '../lib/stores.js';

  let plan = null;
  let enabled = false;
  let maskedKey = '';
  let generating = false;
  let genStatus = '';
  let assets = null;

  const SECTIONS = [
    ['characters', '人物'],
    ['monsters', '怪物'],
    ['scenes', '场景'],
    ['maptiles', '地图块'],
    ['items', '物品'],
  ];
  const KEYS = { characters: 'characters', monsters: 'monsters', scenes: 'scenes', maptiles: 'tiles', items: 'items' };

  async function load() {
    const [cfg, pl, ast] = await Promise.all([j('/api/art/config'), j('/api/art/plan'), j('/api/art/assets')]);
    if (cfg && cfg.ok) {
      enabled = cfg.enabled;
      maskedKey = cfg.config?.api_key_masked || '';
    }
    if (pl && pl.ok) plan = pl.plan;
    if (ast && ast.ok) assets = ast.assets;
  }

  $: if ($refreshTick) load();

  async function generatePlan() {
    generating = true;
    genStatus = '规划生成中（LLM 一次调用，约 1~3 分钟）…';
    await j('/api/art/plan', { method: 'POST' });
    // 轮询直到 plan.json 出现
    let tries = 0;
    const timer = setInterval(async () => {
      tries++;
      const d = await j('/api/art/plan');
      if (d && d.ok && d.plan) {
        clearInterval(timer);
        plan = d.plan;
        genStatus = '规划已就绪';
        generating = false;
        await load();
        toast('素材规划已生成', 'success');
      } else if (tries > 60) {
        clearInterval(timer);
        genStatus = '规划超时，请重试或到 /studio 查看详情';
        generating = false;
      }
    }, 3000);
  }

  async function generateAll() {
    generating = true;
    genStatus = '批量生成中（5 张 sheet，约 3~5 分钟）…';
    const d = await j('/api/art/generate', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ assets: [] }) });
    if (!d || !d.ok) {
      genStatus = d?.error || '提交失败';
      generating = false;
      return;
    }
    const timer = setInterval(async () => {
      const j2 = await j('/api/art/jobs/' + d.job.id);
      if (!j2 || !j2.ok) return;
      const steps = j2.job.steps || [];
      const detail = steps.map((s) => `${s.label}:${s.status === 'done' ? '✔' : s.status === 'failed' ? '✘' : '…'}`).join(' ');
      if (j2.job.status === 'done' || j2.job.status === 'failed') {
        clearInterval(timer);
        genStatus = `完成 ${detail}`;
        generating = false;
        await load();
        toast('素材生成完成', 'success');
      } else {
        genStatus = `生成中 ${detail}`;
      }
    }, 5000);
  }

  function spriteCount(tab) {
    if (!assets || !assets[tab]) return 0;
    return (assets[tab].sprites || []).filter((s) => s.exists).length;
  }
  function totalSprites() {
    return SECTIONS.reduce((acc, [k]) => acc + spriteCount(k), 0);
  }
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🎨</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">
      美术工坊
      <span class="text-xs text-base-content/40 font-normal">世界书 → 像素素材</span>
    </h2>
    <span class="ml-auto badge badge-sm {enabled ? 'badge-success' : 'badge-ghost'}">
      {enabled ? `已配置 ${maskedKey}` : '未配置服务'}
    </span>
  </div>

  {#if plan}
    <div class="text-xs text-base-content/60 mb-2">
      规划：theme <b>{plan.theme}</b> · 人物 {plan.characters?.length || 0} / 怪物 {plan.monsters?.length || 0} / 场景 {plan.scenes?.length || 0} / 地图块 {plan.tiles?.length || 0} / 物品 {plan.items?.length || 0}
      · 已产出素材 <b class="text-success">{totalSprites()}</b> 个
    </div>
    <div class="flex gap-2 mb-2 flex-wrap">
      <button class="btn btn-sm btn-primary" on:click={generateAll} disabled={generating || !enabled}>⚡ 全部生成</button>
      <a class="btn btn-sm btn-ghost" href="/studio" target="_blank">编辑规划 →</a>
    </div>
    {#if genStatus}<div class="text-xs text-primary mb-2">{genStatus}</div>{/if}
    <div class="grid grid-cols-2 gap-1.5 text-xs">
      {#each SECTIONS as [k, zh]}
        <div class="flex justify-between rounded px-2 py-1 bg-base-content/5">
          <span>{zh}</span>
          <span class="text-base-content/50">{spriteCount(k)}/{assets?.[k]?.count || 0}</span>
        </div>
      {/each}
    </div>
  {:else}
    <div class="text-sm text-base-content/40 mb-3">
      还没有素材规划。让 AI 读你的世界书，规划 12 人物 / 8 怪物 / 3 场景 / 16 地图块 / 24 物品的像素素材与提示词。
    </div>
    <button class="btn btn-sm btn-primary" on:click={generatePlan} disabled={generating}>📜 从世界书生成规划</button>
    {#if genStatus}<div class="text-xs text-primary mt-2">{genStatus}</div>{/if}
  {/if}

  {#if !enabled}
    <div class="text-xs text-base-content/40 mt-3 border-t border-base-content/10 pt-2">
      图片生成服务未配置：到 <a class="link" href="/studio" target="_blank">/studio</a> 填 base_url 与 api_key（或设环境变量 GPTIMG_KEY）。
    </div>
  {/if}
</div>
