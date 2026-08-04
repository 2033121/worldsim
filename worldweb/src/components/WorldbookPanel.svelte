<script>
  import { j, escapeHtml } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let content = '';

  async function load() {
    const d = await j('/api/world/worldbook');
    if (d.content !== undefined) content = d.content;
  }

  $: if ($refreshTick) load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">📚</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">世界书</h2>
  </div>
  {#if !content}
    <div class="text-sm text-base-content/40 text-center py-6">（世界书为空）</div>
  {:else}
    <div class="whitespace-pre-wrap text-sm leading-relaxed text-base-content/80 max-h-[68vh] overflow-y-auto p-5 rounded-lg bg-base-200/50 border border-base-content/10">
      {escapeHtml(content)}
    </div>
  {/if}
</div>