// theme.js — 明暗主题切换 + cookie/localStorage 持久化
import { writable } from 'svelte/store';

function initialTheme() {
  try {
    return localStorage.getItem('worldsim.theme') || 'emerald';
  } catch (e) { return 'emerald'; }
}

export const theme = writable(initialTheme());

export function setTheme(t) {
  theme.set(t);
  try { localStorage.setItem('worldsim.theme', t); } catch (e) {}
  applyTheme(t);
}

export function toggleTheme() {
  setTheme(getTheme() === 'emerald' ? 'night' : 'emerald');
}

export function getTheme() {
  let v = 'emerald';
  theme.subscribe(t => v = t)();
  return v;
}

export function applyTheme(t) {
  const el = document.documentElement;
  el.setAttribute('data-theme', t);
}