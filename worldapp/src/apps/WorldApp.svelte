<script>
  // WorldApp — 世界模拟应用（过渡：iframe 嵌入 :48091 现有 WebUI）
  // 后续 Phase 3 将逐步原生迁移，替换为原生 Svelte 页面。
  const WORLD_UI = import.meta.env.VITE_WORLD_UI || 'http://localhost:48091/';
</script>

<div class="h-full w-full">
  <iframe
    src={WORLD_UI}
    class="w-full h-full border-0"
    title="世界模拟控制台"
    sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-modals"
  ></iframe>
</div>