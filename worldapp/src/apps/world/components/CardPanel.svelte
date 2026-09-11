<script>
  // CardPanel — 世界卡：把"一个世界"打包成单文件带走/分享（导出/导入）。
  // 后端：GET /api/world/card[?with_game=1]（zip 下载）；POST /api/worlds/import（multipart 上传）。
  import { toast, refreshAll } from '../lib/stores.js';

  let withGame = false;
  let importing = false;
  let customName = '';

  async function exportCard() {
    // 直接触发浏览器下载（zip）
    window.location.href = '/api/world/card' + (withGame ? '?with_game=1' : '');
    toast('世界卡已开始下载' + (withGame ? '（含游玩进度）' : ''), 'success');
  }

  async function onFile(e) {
    const f = e.target.files?.[0];
    if (!f) return;
    importing = true;
    try {
      const fd = new FormData();
      fd.append('file', f);
      if (customName.trim()) fd.append('name', customName.trim());
      const r = await fetch('/api/worlds/import', { method: 'POST', body: fd });
      const d = await r.json().catch(() => ({ ok: false, error: '非 JSON 响应' }));
      if (d.ok) {
        toast(`世界卡已导入：${d.world}（${d.files} 个文件，已自动选中）`, 'success');
        await refreshAll();
      } else {
        toast('导入失败：' + (d.error || '未知错误'), 'error');
      }
    } catch (err) {
      toast('导入失败：' + err, 'error');
    }
    importing = false;
    e.target.value = '';
  }
</script>

<div class="space-y-3">
  <div class="flex items-center gap-2">
    <span class="text-lg">🃏</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">世界卡 · 分享</h2>
  </div>
  <p class="text-xs text-base-content/60 leading-relaxed">
    把当前世界打包成 <code>.worldcard.zip</code>（世界书 + 专属像素素材 + 可选游玩进度）发给别人；
    对方导入后得到同一个世界继续游玩。不含编年史/记忆/小说（隐私边界——那些是玩出来的）。
  </p>
  <label class="flex items-center gap-2 text-xs cursor-pointer">
    <input type="checkbox" class="checkbox checkbox-xs" bind:checked={withGame} />
    附带游玩进度（game.json：数值/背包/好感一起带走）
  </label>
  <button class="btn btn-primary btn-sm w-full" on:click={exportCard}>📤 导出当前世界卡</button>

  <div class="divider my-1 text-[10px] opacity-40">导入别人的世界卡</div>
  <input class="input input-sm input-bordered w-full" placeholder="可选：自定义新世界名（默认用卡内名字）" bind:value={customName} />
  <label class="btn btn-ghost btn-sm w-full {importing ? 'btn-disabled' : ''}">
    {#if importing}<span class="loading loading-spinner loading-xs"></span>{/if}
    📥 选择 .worldcard.zip 导入
    <input type="file" accept=".zip" class="hidden" on:change={onFile} disabled={importing} />
  </label>
</div>
