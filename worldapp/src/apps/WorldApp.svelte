<script>
  // WorldApp — 世界模拟应用（原生渲染，Phase 3 迁移）
  // 组件与 API 迁自 worldweb，走统一网关同源调用（/api/* → 48091）。
  import Console from './world/Console.svelte';
</script>

<div class="h-full w-full overflow-hidden">
  <Console />
</div>