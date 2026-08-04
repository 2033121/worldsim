<script>
  import { openTab, navigateTo } from '../lib/browser.js';
  import { ROUTES } from '../lib/routes.js';

  // 快捷入口（首屏）
  const quickLinks = [
    { address: 'world', icon: '🌍', title: '世界模拟', desc: '题材研究 / 编年史 / 决策 / 小说 / 实体 / 世界书', color: 'from-sky-500 to-blue-600' },
    { address: 'novel', icon: '📖', title: '小说创作', desc: '项目 / 大纲 / 写作 / 记忆 / 关系 / 技能 / 伏笔', color: 'from-violet-500 to-purple-600' },
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
        class="card card-body bg-gradient-to-br {q.color} text-white text-left hover:shadow-xl hover:-translate-y-0.5 transition-all cursor-pointer"
        on:click={() => navigateTo(q.address)}
      >
        <div class="text-3xl mb-2">{q.icon}</div>
        <div class="text-xl font-bold">{q.title}</div>
        <div class="text-sm opacity-80">{q.desc}</div>
        <div class="mt-3 text-sm opacity-90 flex items-center gap-1">进入 <span>→</span></div>
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