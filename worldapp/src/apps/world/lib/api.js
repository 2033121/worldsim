// API 请求封装：统一返回 JSON，错误捕获
export async function j(url, opt) {
  try {
    const r = await fetch(url, opt);
    return await r.json();
  } catch (e) {
    return { ok: false, error: '网络错误: ' + e.message };
  }
}

export const MODELS = [
  'deepseek-v4-flash', 'deepseek-v4-flash-0731', 'deepseek-v4-pro',
  'glm-5', 'glm-5.1', 'glm-5.2',
  'qwen3.7-max', 'qwen3.8-max',
  'kimi-k2.5', 'kimi-k2.6', 'kimi-k2.7-code',
  'minimax-m2.5', 'minimax-m2.7', 'mimo-v2.5-pro'
];