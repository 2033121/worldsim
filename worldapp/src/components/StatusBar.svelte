<script>
  import { statusText } from '../lib/browser.js';
</script>

<div class="flex items-center gap-3 px-3 py-1 text-xs border-t border-base-300/50 bg-base-200/40 text-base-content/60 shrink-0">
  <span class="flex items-center gap-1">
    <span class="w-2 h-2 rounded-full bg-success inline-block"></span>
    服务正常
  </span>
  <span class="flex-1 truncate text-right">{$statusText}</span>
</div>