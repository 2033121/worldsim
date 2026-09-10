<script>
  import { worldState } from '../lib/stores.js';
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🏙</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">全局事件</h2>
  </div>
  {#if $worldState?.world_level?.global_events?.length}
    <div class="text-sm leading-relaxed text-base-content/80 space-y-1">
      {#each $worldState.world_level.global_events as ev}
        <div>· {ev}</div>
      {/each}
    </div>
  {:else}
    <div class="text-sm text-base-content/40">无</div>
  {/if}
</div>