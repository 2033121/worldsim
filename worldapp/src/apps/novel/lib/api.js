import { getLocale, translateServerMessage } from './i18n/index.js';

// 统一前端网关前缀：小说服务在 :48092 上走 /api/novel/*（→ :48090 /api/*）。
// 组件内 url 形如 '/api/projects'（带前导 /api），故拼接时去掉前导 /api，
// 避免产生 '/api/novel/api/...' 的重复前缀。
const PREFIX = '/api/novel';

export async function api(method, url, body) {
  const locale = getLocale();
  const opts = {
    method,
    headers: {
      'Content-Type': 'application/json',
      'X-UI-Locale': locale,
      'Accept-Language': locale === 'en' ? 'en-US,en;q=0.9' : 'zh-CN,zh;q=0.9',
    },
  };
  if (body) opts.body = JSON.stringify(body);
  const path = url.startsWith('/api') ? url.slice(4) : url;
  const r = await fetch(PREFIX + path, opts);
  const data = await r.json();
  if (!r.ok) {
    const raw = data.error || 'Request failed';
    // Backend mostly responds in Chinese today; translate known strings on the client.
    throw new Error(translateServerMessage(raw, locale));
  }
  return data;
}
