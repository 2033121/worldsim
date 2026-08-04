<script>
  import { worldState, currentWorld } from '../lib/stores.js';
  import { escapeHtml } from '../lib/api.js';
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">📊</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">世界状态</h2>
  </div>
  {#if !$worldState}
    <div class="text-sm text-base-content/40 text-center py-6 border border-dashed border-base-content/20 rounded-lg">初始化后显示</div>
  {:else}
    <div class="space-y-1.5 text-sm">
      {#if $currentWorld}
        <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
          <span class="text-base-content/50">🌐 世界</span><span class="font-medium">{escapeHtml($currentWorld)}</span>
        </div>
      {/if}
      <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
        <span class="text-base-content/50">📅 天数</span><span class="font-medium">Day {$worldState.day}</span>
      </div>
      <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
        <span class="text-base-content/50">⚡ 张力</span><span class="font-medium text-warning">{($worldState.world_level?.tension || 0).toFixed(2)}</span>
      </div>
      <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
        <span class="text-base-content/50">🌦 天气</span><span class="font-medium">{escapeHtml($worldState.weather || '-')}</span>
      </div>
      {#if $worldState.entities}
        {#each Object.entries($worldState.entities) as [k, e]}
          {#if e.extra?.role === 'protagonist'}
            <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
              <span class="text-base-content/50">🧍 主角</span><span class="font-medium">{escapeHtml(k)} · {escapeHtml(e.job || '-')}</span>
            </div>
            <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
              <span class="text-base-content/50">💰 金钱</span><span class="font-medium">¥{e.money ?? '-'}</span>
            </div>
            <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
              <span class="text-base-content/50">❤️ 健康</span><span class="font-medium">{e.health ?? '-'}</span>
            </div>
            <div class="flex justify-between border-b border-dashed border-base-content/10 py-1">
              <span class="text-base-content/50">📍 位置</span><span class="font-medium">{escapeHtml(e.location || '-')}</span>
            </div>
          {/if}
        {/each}
      {/if}
    </div>
  {/if}
</div>