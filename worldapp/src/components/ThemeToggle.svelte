<script>
  import { theme, toggleTheme } from '../lib/theme.js';
</script>

<button
  class="btn btn-ghost btn-circle btn-sm"
  on:click={toggleTheme}
  title="切换主题"
>
  {#if $theme === 'emerald'}🌙{:else}☀️{/if}
</button>