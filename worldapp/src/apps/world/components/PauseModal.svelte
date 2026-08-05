<script>
  let visible = false;
  let data = null;

  export function show(d) {
    data = d;
    visible = true;
  }
  function hide() { visible = false; }
</script>

{#if visible && data}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center" on:click={hide}>
    <div class="paper rounded-2xl p-6 w-[92%] max-w-md max-h-[82vh] overflow-y-auto" on:click|stopPropagation>
      <div class="flex items-center gap-2 mb-4">
        <span class="text-xl cloud-icon">⚡</span>
        <h3 class="text-lg font-semibold text-warning">抉择点</h3>
        <div class="flex-1"></div>
        <button class="btn btn-ghost btn-sm btn-circle" on:click={hide}>✕</button>
      </div>

      <div class="text-sm leading-relaxed text-base-content/85">
        {data.results?.[data.results.length - 1]?.pauseMsg || '发生重大事件'}
      </div>

      {#if data.results?.[data.results.length - 1]?.events?.[0]?.options?.length}
        <div class="mt-4 space-y-2">
          {#each data.results[data.results.length - 1].events[0].options as o, i}
            <div class="rounded-xl bg-base-200/50 border border-base-content/15 p-4 text-sm cursor-pointer hover:border-warning/50 transition-all" on:click={hide}>
              {i + 1}. {o}
            </div>
          {/each}
        </div>
      {/if}

      <div class="mt-4">
        <button class="btn btn-primary btn-sm w-full" on:click={hide}>继续模拟</button>
      </div>
    </div>
  </div>
{/if}