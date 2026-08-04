// routes.js — 地址 → 页面组件映射
// address 是浏览器外壳导航键；component 为该地址渲染的页面。
import HomePage from '../pages/HomePage.svelte';
import WorldApp from '../apps/WorldApp.svelte';
import NovelApp from '../apps/NovelApp.svelte';

export const ROUTES = {
  home: { title: '首页', icon: '🏠', hint: '最近项目、世界与快捷入口', component: HomePage },
  world: { title: '世界模拟', icon: '🌍', hint: '世界模拟控制台（题材研究/编年史/决策/小说/实体…）', component: WorldApp },
  novel: { title: '小说创作', icon: '📖', hint: '小说创作流水线（项目/大纲/写作/记忆/关系/技能…）', component: NovelApp },
};

// 默认启动地址
export const DEFAULT_HOME = 'home';