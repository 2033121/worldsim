import { writable } from 'svelte/store';

// 当前选中世界
export const currentWorld = writable('');
// 当前已加载世界列表
export const worlds = writable([]);
// 主角名
export const heroName = writable('');
// 世界状态
export const worldState = writable(null);
// 上次模拟结果（用于今日对话/事件）
export const lastResult = writable(null);
// 后台循环状态
export const loopInfo = writable(null);
// Toast 消息队列
export const toasts = writable([]);
// 当前激活 tab
export const activeTab = writable('chronicle');
// 全局刷新计数（供组件监听触发重新加载）
export const refreshTick = writable(0);

let toastId = 0;
export function toast(msg, type = 'info') {
  const id = ++toastId;
  toasts.update((list) => [...list, { id, msg, type }]);
  setTimeout(() => {
    toasts.update((list) => list.filter((t) => t.id !== id));
  }, 2600);
}

export function refreshAll() {
  refreshTick.update((n) => n + 1);
}