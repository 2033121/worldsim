// api.js — 统一 API 网关调用
// 由于网关在 :48092 上：小说走 /api/novel/*（→48090），世界走其真实路径（→48091）。
// 前端统一走同源相对路径，无跨域。
import { writable, get } from 'svelte/store';

export const toasts = writable([]);
let toastSeq = 0;
export function toast(msg, type = 'info') {
  const id = 't' + (toastSeq++);
  toasts.update(list => [...list, { id, msg, type }]);
  setTimeout(() => toasts.update(list => list.filter(t => t.id !== id)), 3500);
}

// 统一请求：url 以 /api/novel 或 /api/world 开头，走网关
export async function j(url, opt = {}) {
  try {
    opt.headers = opt.headers || {};
    opt.headers['Content-Type'] = opt.headers['Content-Type'] || 'application/json';
    const r = await fetch(url, opt);
    const data = await r.json().catch(() => ({}));
    if (!r.ok) {
      const err = data.error || ('请求失败 ' + r.status);
      return { ok: false, error: err };
    }
    return data;
  } catch (e) {
    return { ok: false, error: '网络错误: ' + e.message };
  }
}

// 小说服务 API（网关 /api/novel/* → 48090 /api/*）
export const novelApi = {
  get: (p) => j('/api/novel' + p),
  post: (p, body) => j('/api/novel' + p, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  del: (p) => j('/api/novel' + p, { method: 'DELETE' }),
};

// 世界模拟 API（真实路径 → 48091）
export const worldApi = {
  get: (p) => j('/api' + p),
  post: (p, body) => j('/api' + p, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  del: (p) => j('/api' + p, { method: 'DELETE' }),
};

export const MODELS = [
  'deepseek-v4-flash', 'deepseek-v4-flash-0731', 'deepseek-v4-pro',
  'glm-5', 'glm-5.1', 'glm-5.2',
  'qwen3.7-max', 'qwen3.8-max',
  'kimi-k2.5', 'kimi-k2.6', 'kimi-k2.7-code',
  'minimax-m2.5', 'minimax-m2.7', 'mimo-v2.5-pro'
];

// 世界模拟面板刷新 tick（供各面板自动刷新）
export const refreshTick = writable(0);
export function refreshAll() { refreshTick.update(n => n + 1); }