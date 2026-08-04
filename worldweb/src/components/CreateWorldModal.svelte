<script>
  import { j } from '../lib/api.js';
  import { toast, refreshAll } from '../lib/stores.js';

  let visible = false;
  let newName = '';
  let newTheme = '';
  let newDesc = '';
  let newWorldbook = '';
  let newDirection = ''; // 研究结果引导的世界书方向
  let themed = false; // 是否来自研究结果
  let themes = [];

  export function show(opts = {}) {
    visible = true;
    themed = !!opts.direction;
    newDirection = opts.direction || '';
    if (!opts.direction) {
      newName = '';
      newDesc = '';
      newTheme = '';
      loadThemes();
    }
  }
  function hide() { visible = false; }
  function switchManual() {
    themed = false;
    newDirection = '';
    loadThemes();
  }

  async function loadThemes() {
    const d = await j('/api/worldbooks/themes');
    themes = d.themes || [];
  }

  async function create() {
    const name = newName.trim();
    if (!name) { toast('请输入世界名称', 'error'); return; }
    const body = { name };
    if (themed && newDirection) {
      body.worldbook_direction = newDirection;
    } else {
      if (newTheme) body.theme = newTheme;
      if (newDesc) body.desc = newDesc.trim();
      if (newWorldbook) body.worldbook = newWorldbook.trim();
    }
    hide();
    const d = await j('/api/worlds/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (d.ok) { toast('世界已创建：' + name, 'success'); refreshAll(); }
    else toast(d.error || '创建失败', 'error');
  }
</script>

{#if visible}
  <div class="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center" on:click={hide}>
    <div class="paper rounded-2xl p-6 w-[92%] max-w-md max-h-[82vh] overflow-y-auto" on:click|stopPropagation>
      <div class="flex items-center gap-2 mb-4">
        <span class="text-xl cloud-icon">🌐</span>
        <h3 class="text-lg font-semibold text-primary">新建世界</h3>
        <div class="flex-1"></div>
        <button class="btn btn-ghost btn-sm btn-circle" on:click={hide}>✕</button>
      </div>

      <label class="label py-0 text-xs text-base-content/60">世界名称</label>
      <input class="input input-sm input-bordered w-full mb-2" placeholder="如：青岚界·灵剑山" bind:value={newName} />

      {#if themed}
        <div class="alert alert-info text-sm mb-2">🔬 已载入研究结果的世界书方向，将据此生成完整世界书。</div>
        <label class="label py-0 text-xs text-base-content/60">一句话设定（可选，研究引导模式下补充润色）</label>
        <input class="input input-sm input-bordered w-full mb-2" placeholder="如：灵气复苏三百年后…" bind:value={newDesc} />
        <div class="text-right mb-2"><button class="link link-primary text-xs" on:click={switchManual}>改用手动/主题包模式</button></div>
      {:else}
        <label class="label py-0 text-xs text-base-content/60">🎨 主题包（选它 → LLM 自动生成完整世界书）</label>
        <select class="select select-sm select-bordered w-full mb-2" bind:value={newTheme}>
          <option value="">（手动指定世界书）</option>
          {#each themes as t}
            <option value={t}>{t}</option>
          {/each}
        </select>

        <label class="label py-0 text-xs text-base-content/60">一句话设定（可选，主题包模式下 LLM 按此生成）</label>
        <input class="input input-sm input-bordered w-full mb-2" placeholder="如：灵气复苏三百年后的青岚界…" bind:value={newDesc} />

        <label class="label py-0 text-xs text-base-content/60">世界书（可选，worldbooks/ 池，手动模式用）</label>
        <input class="input input-sm input-bordered w-full mb-2" placeholder="如：临江市·都市怪谈（留空用默认）" bind:value={newWorldbook} />
      {/if}

      <div class="flex gap-2 mt-4">
        <button class="btn btn-primary btn-sm flex-1" on:click={create}>创建并进入</button>
        <button class="btn btn-ghost btn-sm" on:click={hide}>取消</button>
      </div>
    </div>
  </div>
{/if}