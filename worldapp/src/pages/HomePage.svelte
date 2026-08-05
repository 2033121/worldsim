<script>
  import { openTab, navigateTo } from '../lib/browser.js';
  import { ROUTES } from '../lib/routes.js';

  // 快捷入口（首屏）
  const quickLinks = [
    { address: 'world', icon: '🌍', title: '世界模拟', desc: '题材研究 / 编年史 / 决策 / 小说 / 实体 / 世界书', accent: 'from-amber-500/15 to-orange-500/5', seal: '印' },
    { address: 'novel', icon: '📖', title: '小说创作', desc: '项目 / 大纲 / 写作 / 记忆 / 关系 / 技能 / 伏笔', accent: 'from-rose-500/15 to-red-500/5', seal: '文' },
  ];
</script>

<div class="page-enter max-w-5xl mx-auto p-8">
  <div class="text-center pt-8 pb-10">
    <div class="text-5xl mb-3">🌐</div>
    <h1 class="text-3xl font-bold">WorldSim 统一前端</h1>
    <p class="text-base-content/60 mt-2">多世界模拟 · AI 世界演绎 · 一站式创作平台</p>
    <p class="text-sm opacity-50 mt-1">浏览器式导航：多标签 / 前进后退 / 书签 / 命令面板 (Ctrl+K)</p>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
    {#each quickLinks as q (q.address)}
      <button
        class="paper card card-body bg-gradient-to-br {q.accent} text-left hover:shadow-xl hover:-translate-y-0.5 transition-all cursor-pointer group"
        on:click={() => navigateTo(q.address)}
      >
        <div class="flex items-center justify-between">
          <div class="text-3xl mb-2">{q.icon}</div>
          <span class="seal bg-error/10 text-error px-2 py-0.5 rounded text-lg">{q.seal}</span>
        </div>
        <div class="text-xl font-bold text-base-content">{q.title}</div>
        <div class="text-sm text-base-content/60">{q.desc}</div>
        <div class="mt-3 text-sm text-primary flex items-center gap-1">进入 <span class="group-hover:translate-x-1 transition-transform">→</span></div>
      </button>
    {/each}
  </div>

  <div class="mt-8">
    <div class="text-sm font-semibold opacity-60 mb-2">📌 可用页面</div>
    <div class="flex flex-wrap gap-2">
      {#each Object.keys(ROUTES) as addr}
        <button class="badge badge-outline badge-lg py-3 cursor-pointer hover:badge-primary" on:click={() => openTab(addr)}>
          {ROUTES[addr].icon} {ROUTES[addr].title}
        </button>
      {/each}
    </div>
  </div>
</div>