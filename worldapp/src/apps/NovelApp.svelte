<script>
  // NovelApp — 小说创作应用（原生渲染，Phase 4 迁移）
  // 组件与 API 迁自 frontend，走统一网关 /api/novel/*（→ :48090）。
  import Console from './novel/Console.svelte';
</script>

<div class="h-full w-full overflow-hidden">
  <Console />
</div>