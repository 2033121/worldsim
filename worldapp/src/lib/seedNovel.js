// seedNovel.js — 跨应用「世界播种小说」请求协调
// 世界应用（apps/world）点击「据此生成小说」时写入 { world_id, ts } 并跳到小说 tab；
// 小说应用（apps/novel）挂载后读取并消费，打开播种面板（自动带入 world_id）。
import { writable } from 'svelte/store';

export const seedNovelRequest = writable(null); // { world_id, ts }