<script>
  import { openTab, navigateTo } from '../lib/browser.js';
  import { ROUTES } from '../lib/routes.js';

  // 两大核心应用（首屏主入口）
  const apps = [
    {
      address: 'world',
      icon: '🌍',
      seal: '境',
      title: '世界模拟',
      tagline: '多世界 · AI 演绎 · 自动推进',
      desc: '题材研究、编年史、决策、小说、实体、世界书一站管理，让每个世界自己生长。',
      features: ['题材研究 → 世界书方向', '编年史 / 决策 / 事件推进', '实体 & 世界书管理', '联网搜索 + 参考资料注入'],
      accent: 'from-amber-500/15 to-orange-500/5',
      sealCls: 'bg-amber-500/10 text-amber-700',
    },
    {
      address: 'novel',
      icon: '📖',
      seal: '文',
      title: '小说创作',
      tagline: '项目 · 大纲 · 写作 · 全流程',
      desc: '从选题到大纲、逐章创作、事实核查、伏笔管理，AI 助力长篇故事创作。',
      features: ['项目 / 大纲 / 卷骨架', '逐章创作 + 事实核查', '正文整章编辑 & 块级编辑', '记忆 / 关系 / 技能 / 伏笔'],
      accent: 'from-rose-500/15 to-red-500/5',
      sealCls: 'bg-rose-500/10 text-rose-700',
    },
  ];

  // 能力亮点（功能标签）
  const highlights = ['多标签页', '前进 / 后退', '书签收藏', '命令面板 Ctrl+K', '明暗主题'];
</script>

<div class="page-enter max-w-5xl mx-auto p-6 md:p-10">
  <!-- Hero -->
  <div class="text-center pt-6 pb-4">
    <div class="inline-flex items-center gap-3 mb-4">
      <span class="seal bg-primary/10 text-primary px-3 py-1.5 rounded-md text-2xl">演</span>
      <span class="scroll-rule w-16"></span>
      <span class="text-4xl">🌐</span>
      <span class="scroll-rule w-16"></span>
      <span class="seal bg-accent/10 text-accent px-3 py-1.5 rounded-md text-2xl">创</span>
    </div>
    <h1 class="text-4xl md:text-5xl font-bold tracking-wide">WorldSim 统一前端</h1>
    <p class="text-base-content/60 mt-3 text-lg">多世界模拟 · AI 世界演绎 · 一站式创作平台</p>
    <div class="flex flex-wrap justify-center gap-2 mt-5">
      {#each highlights as h}
        <span class="badge badge-outline badge-ghost badge-lg py-3 px-3">{h}</span>
      {/each}
    </div>
  </div>

  <!-- 两大核心入口 -->
  <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-6">
    {#each apps as a (a.address)}
      <button
        class="paper card card-body bg-gradient-to-br {a.accent} text-left hover:shadow-xl hover:-translate-y-1 transition-all duration-200 cursor-pointer group p-6"
        on:click={() => navigateTo(a.address)}
      >
        <div class="flex items-center justify-between">
          <div class="text-4xl">{a.icon}</div>
          <span class="seal {a.sealCls} px-2.5 py-1 rounded text-lg">{a.seal}</span>
        </div>
        <div class="mt-3">
          <div class="text-2xl font-bold text-base-content flex items-center gap-2">{a.title}</div>
          <div class="text-sm text-primary/80 mt-0.5">{a.tagline}</div>
        </div>
        <p class="text-sm text-base-content/60 mt-2 leading-relaxed">{a.desc}</p>
        <ul class="mt-3 space-y-1.5">
          {#each a.features as f (f)}
            <li class="flex items-center gap-2 text-sm text-base-content/70">
              <span class="text-success shrink-0">✓</span>{f}
            </li>
          {/each}
        </ul>
        <div class="mt-4 text-sm text-primary font-medium flex items-center gap-1">
          进入 <span class="group-hover:translate-x-1 transition-transform">→</span>
        </div>
      </button>
    {/each}
  </div>

  <!-- 全部可用页面 -->
  <div class="mt-8">
    <div class="flex items-center gap-3 mb-3">
      <span class="scroll-rule flex-1"></span>
      <div class="text-sm font-semibold opacity-70">📌 全部页面</div>
      <span class="scroll-rule flex-1"></span>
    </div>
    <div class="flex flex-wrap justify-center gap-2">
      {#each Object.keys(ROUTES) as addr}
        <button
          class="btn btn-ink btn-sm"
          on:click={() => openTab(addr)}
        >
          <span>{ROUTES[addr].icon}</span>
          <span>{ROUTES[addr].title}</span>
          <span class="text-xs opacity-50">{ROUTES[addr].hint}</span>
        </button>
      {/each}
    </div>
  </div>

  <!-- 底部提示 -->
  <div class="text-center mt-10 text-xs text-base-content/40">
    在顶栏地址栏输入地址、按 <kbd class="kbd kbd-xs">Ctrl</kbd> + <kbd class="kbd kbd-xs">K</kbd> 打开命令面板，或拖拽标签页管理多任务
  </div>
</div>