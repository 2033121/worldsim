<script>
  import { j } from '../lib/api.js';
  import { refreshTick, toast } from '../lib/stores.js';

  let attachments = [];
  let refsLen = 0;
  let uploading = false;
  let fileInput;

  async function load() {
    const d = await j('/api/world/attach');
    attachments = d.attachments || [];
    const r = await j('/api/world/attach/refs');
    refsLen = (r.refs || '').length;
  }

  async function upload() {
    const f = fileInput && fileInput.files && fileInput.files[0];
    if (!f) { toast('请先选择文件', 'error'); return; }
    uploading = true;
    const fd = new FormData();
    fd.append('file', f);
    const d = await j('/api/world/attach/upload', { method: 'POST', body: fd });
    uploading = false;
    if (d && d.ok) {
      toast('📎 已上传《' + (d.attachment.name || '') + '》', 'success');
      if (fileInput) fileInput.value = '';
      load();
    } else {
      toast('上传失败：' + ((d && d.error) || ''), 'error');
    }
  }

  async function remove(name) {
    if (!confirm(`删除附件《${name}》？删除后世界参考资料将不再包含它。`)) return;
    const d = await j('/api/world/attach/' + encodeURIComponent(name), { method: 'DELETE' });
    if (d && d.ok) {
      toast('🗑 已删除《' + name + '》', 'success');
      load();
    } else {
      toast('删除失败：' + ((d && d.error) || ''), 'error');
    }
  }

  function fmtSize(n) {
    if (n >= 1048576) return (n / 1048576).toFixed(1) + ' MB';
    if (n >= 1024) return (n / 1024).toFixed(1) + ' KB';
    return n + ' B';
  }

  $: if ($refreshTick) load();
  load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">📎</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">世界参考资料</h2>
    {#if refsLen > 0}
      <span class="badge badge-ghost badge-xs ml-auto">注入 {refsLen} 字</span>
    {/if}
  </div>
  <p class="text-[11px] text-base-content/40 mb-2">
    上传设定/事实文本（txt / md / json / csv / docx 等），聚合后注入所有 Agent 上下文，剧情必须遵循。
  </p>

  <div class="flex gap-2">
    <input
      bind:this={fileInput}
      type="file"
      class="file-input file-input-sm file-input-bordered flex-1"
      />
    <button class="btn btn-primary btn-sm" on:click={upload} disabled={uploading}>
      {uploading ? '上传中…' : '上传'}
    </button>
  </div>

  <div class="mt-2 max-h-44 overflow-y-auto space-y-1">
    {#if !attachments.length}
      <div class="text-[11px] text-base-content/40">暂无附件（上传后自动作为世界参考资料注入）</div>
    {:else}
      {#each attachments as a (a.name)}
        <div class="flex items-center gap-2 py-1 border-b border-dashed border-base-content/10 text-xs">
          <span class="text-base-content/70 truncate flex-1">{a.name}</span>
          {#if a.extractable}
            <span class="badge badge-success badge-xs shrink-0">已注入</span>
          {:else}
            <span class="badge badge-warning badge-xs shrink-0" title="该格式暂不支持直接提取，仅保存文件">未提取</span>
          {/if}
          <span class="text-[10px] text-base-content/30 w-12 text-right shrink-0">{fmtSize(a.size)}</span>
          <button class="btn btn-ghost btn-xs px-1 shrink-0" on:click={() => remove(a.name)} title="删除">🗑</button>
        </div>
      {/each}
    {/if}
  </div>
</div>