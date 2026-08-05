<script>
  import { j } from '../lib/api.js';
  import { refreshTick } from '../lib/stores.js';

  let thinking = '';

  async function load() {
    const d = await j('/api/world/sim/thinking');
    if (d.thinking !== undefined) thinking = d.thinking || '';
  }

  $: if ($refreshTick) load();
</script>

<div class="paper rounded-xl p-4">
  <div class="flex items-center gap-2 mb-2">
    <span class="text-lg cloud-icon">🧠</span>
    <h2 class="text-sm font-semibold text-primary tracking-widest">主角内心 <span class="text-xs text-base-content/40 font-normal">三问决策</span></h2>
  </div>
  <div class="text-sm leading-loose text-info whitespace-pre-wrap max-h-52 overflow-y-auto p-3 rounded-lg bg-primary/5 border border-primary/20 border-l-4">
    {thinking || '尚未运行…'}
  </div>
</div>