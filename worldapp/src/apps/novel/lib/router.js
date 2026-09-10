import { writable } from 'svelte/store';

// 统一前端内的小说路由：不依赖 window.location.hash（浏览器外壳地址在 store 中），
// 直接用 store 管理当前页面。初始 'config'。
export const currentPage = writable('config');

export function navigate(page) {
  currentPage.set(page);
}