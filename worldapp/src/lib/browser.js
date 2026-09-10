// browser.js — 浏览器式导航外壳的状态模型
// 维护：多标签(tabs)、当前标签(activeTabId)、每个标签的历史栈(tabNavs)、书签(bookmarks)。
// 地址(address)是应用内路由键（如 'home' / 'world' / 'novel'），由 routes.js 解析为页面组件。
import { writable, get } from 'svelte/store';
import { ROUTES } from './routes.js';

export const tabs = writable([]); // [{id,title,icon,address}]
export const activeTabId = writable(null);
export const bookmarks = writable([]); // [{id,title,icon,address}]
// tabNavs: { [tabId]: { stack:string[], index:number } }
const tabNavs = writable({});
export const commandOpen = writable(false);

// 状态栏文本（当前地址描述，供 StatusBar 显示）
export const statusText = writable('');

// ---------- 本地持久化 ----------
const LS_KEY = 'worldsim.browser';
function persist() {
  try {
    localStorage.setItem(LS_KEY, JSON.stringify({
      tabs: get(tabs),
      activeTabId: get(activeTabId),
      bookmarks: get(bookmarks),
    }));
  } catch (e) { /* ignore */ }
}
export function restore() {
  try {
    const raw = localStorage.getItem(LS_KEY);
    if (!raw) return;
    const d = JSON.parse(raw);
    if (Array.isArray(d.tabs) && d.tabs.length) {
      tabs.set(d.tabs);
      if (d.activeTabId && d.tabs.some(t => t.id === d.activeTabId)) {
        activeTabId.set(d.activeTabId);
      } else {
        activeTabId.set(d.tabs[0].id);
      }
    }
    if (Array.isArray(d.bookmarks)) bookmarks.set(d.bookmarks);
  } catch (e) { /* ignore */ }
}

let seq = 0;
function genId() { return 'tab' + (Date.now().toString(36)) + (seq++).toString(36); }

// ---------- 地址解析 ----------
export function resolveRoute(address) {
  return ROUTES[address] || ROUTES.home;
}

// ---------- Tab 操作 ----------
export function openTab(address, opts = {}) {
  const a = ROUTES[address] ? address : 'home';
  const meta = ROUTES[a];
  const id = genId();
  tabs.update(list => [...list, { id, title: meta.title, icon: meta.icon, address: a }]);
  tabNavs.update(m => ({ ...m, [id]: { stack: [a], index: 0 } }));
  activeTabId.set(id);
  if (opts.activate !== false) persist();
}
export function closeTab(id) {
  const list = get(tabs);
  const idx = list.findIndex(t => t.id === id);
  if (idx < 0) return;
  const next = list.filter(t => t.id !== id);
  tabs.set(next);
  tabNavs.update(m => { const { [id]: _drop, ...rest } = m; return rest; });
  if (get(activeTabId) === id) {
    // 激活相邻标签
    const neighbor = next[Math.min(idx, next.length - 1)];
    activeTabId.set(neighbor ? neighbor.id : null);
  }
  if (next.length === 0) {
    // 全部关闭：新建首页标签
    openTab('home');
    return;
  }
  persist();
}
export function switchTab(id) {
  if (get(tabs).some(t => t.id === id)) {
    activeTabId.set(id);
    persist();
  }
}

// ---------- 导航（当前标签） ----------
export function navigateTo(address, opts = {}) {
  const a = ROUTES[address] ? address : 'home';
  const meta = ROUTES[a];
  const id = get(activeTabId);
  if (!id) { openTab(a); return; }
  tabs.update(list => list.map(t => t.id === id ? { ...t, address: a, title: meta.title, icon: meta.icon } : t));
  tabNavs.update(m => {
    const nav = m[id] || { stack: [], index: -1 };
    const stack = nav.stack.slice(0, nav.index + 1);
    if (opts.push !== false) {
      stack.push(a);
      return { ...m, [id]: { stack, index: stack.length - 1 } };
    }
    // replace：不新增历史
    const s2 = stack.slice(0, Math.max(0, stack.length - 1));
    s2.push(a);
    return { ...m, [id]: { stack: s2, index: s2.length - 1 } };
  });
  persist();
}

export function canGoBack() {
  const id = get(activeTabId);
  const nav = get(tabNavs)[id];
  return !!nav && nav.index > 0;
}
export function canGoForward() {
  const id = get(activeTabId);
  const nav = get(tabNavs)[id];
  return !!nav && nav.index < nav.stack.length - 1;
}
export function goBack() {
  const id = get(activeTabId);
  if (!id) return;
  tabNavs.update(m => {
    const nav = m[id];
    if (!nav || nav.index <= 0) return m;
    const index = nav.index - 1;
    const address = nav.stack[index];
    tabs.update(list => list.map(t => t.id === id ? { ...t, ...routeMeta(address) } : t));
    persist();
    return { ...m, [id]: { ...nav, index } };
  });
}
export function goForward() {
  const id = get(activeTabId);
  if (!id) return;
  tabNavs.update(m => {
    const nav = m[id];
    if (!nav || nav.index >= nav.stack.length - 1) return m;
    const index = nav.index + 1;
    const address = nav.stack[index];
    tabs.update(list => list.map(t => t.id === id ? { ...t, ...routeMeta(address) } : t));
    persist();
    return { ...m, [id]: { ...nav, index } };
  });
}
function routeMeta(address) {
  const meta = ROUTES[address] || ROUTES.home;
  return { address, title: meta.title, icon: meta.icon };
}

// 当前活跃标签（供 App 渲染用）
import { derived } from 'svelte/store';
export const activeTab = derived([tabs, activeTabId], ([$tabs, $id]) =>
  $tabs.find(t => t.id === $id) || null
);

// ---------- 书签 ----------
export function addBookmark(address) {
  const a = ROUTES[address] ? address : 'home';
  const meta = ROUTES[a];
  if (get(bookmarks).some(b => b.address === a)) return;
  bookmarks.update(list => [...list, { id: genId(), title: meta.title, icon: meta.icon, address: a }]);
  persist();
}
export function removeBookmark(address) {
  bookmarks.update(list => list.filter(b => b.address !== address));
  persist();
}
export function isBookmarked(address) {
  return get(bookmarks).some(b => b.address === address);
}

// ---------- 命令面板候选 ----------
export const commandItems = Object.keys(ROUTES).map(a => ({
  address: a,
  title: ROUTES[a].title,
  icon: ROUTES[a].icon,
  hint: ROUTES[a].hint || '',
}));