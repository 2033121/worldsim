<script>
  // NovelApp — 小说创作应用（过渡：iframe 嵌入 :48090 现有 WebUI）
  // 后续 Phase 4 将逐步原生迁移，替换为原生 Svelte 页面。
  const NOVEL_UI = import.meta.env.VITE_NOVEL_UI || 'http://localhost:48090/';
</script>

<div class="h-full w-full">
  <iframe
    src={NOVEL_UI}
    class="w-full h-full border-0"
    title="小说创作流水线"
    sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-modals"
  ></iframe>
</div>