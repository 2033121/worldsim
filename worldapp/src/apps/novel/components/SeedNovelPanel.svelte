<script>
  // SeedNovelPanel — 从已模拟世界播种小说项目
  // 世界列表取自世界模拟服务（worldApi /worlds）；播种调用 /api/world/seed-novel（网关直连小说服务 48090）。
  import { onMount } from 'svelte';
  import { worldApi } from '../../../lib/api.js';
  import { api } from '../lib/api.js';
  import {
    currentProject, progress, config, settings, chatSessions, currentChatSession, projectLanguage, addToast,
  } from '../lib/stores.js';
  import { navigate } from '../lib/router.js';
  import { applyDefaultLocale } from '../lib/i18n/index.js';

  let visible = false;
  let worldList = [];
  let selectedWorld = '';   // 下拉选中的世界名
  let manualWorld = '';     // 手动输入的 world_id
  let projectName = '';
  let language = 'zh';
  let seeding = false;
  let err = '';
  let result = null; // SeedResult

  // 世界名（下拉选择或手动输入）
  $: worldId = selectedWorld || manualWorld.trim();

  export function show(world_id = '') {
    visible = true;
    err = '';
    result = null;
    if (world_id) {
      if (worldList.some((w) => w.name === world_id)) {
        selectedWorld = world_id;
        manualWorld = '';
      } else {
        selectedWorld = '';
        manualWorld = world_id;
      }
    }
  }
  function hide() { visible = false; }

  onMount(async () => {
    const d = await worldApi.get('/worlds');
    if (d && !d.ok && d.error) return;
    worldList = (d && d.worlds) || [];
  });

  function mapError(raw) {
    if (!raw) return '播种失败';
    if (raw === 'project_exists' || raw.includes('项目已存在')) return '项目已存在，请换一个项目名';
    if (raw === 'missing_project_name') return '请输入项目名称';
    if (raw === 'project_name_invalid_chars') return '项目名包含非法字符（不能含 / \\ : * ? " < > |）';
    if (raw === 'world_id_required') return '请选择或输入世界';
    if (raw === 'invalid_json') return '参数错误';
    return raw;
  }

  async function seed() {
    err = '';
    if (!worldId) { err = '请选择或输入世界'; return; }
    const name = projectName.trim();
    if (!name) { err = '请输入项目名称'; return; }
    seeding = true;
    const res = await worldApi.post('/world/seed-novel', {
      world_id: worldId,
      project_name: name,
      language,
    });
    seeding = false;
    if (res && res.ok === false) { err = mapError(res.error); return; }
    if (!res || !res.project_name) { err = '播种失败：返回数据异常'; return; }
    result = res;
    addToast('✅ 已从世界「' + (res.world_name || worldId) + '」播种小说「' + res.project_name + '」', 'success');
  }

  // 进入该小说项目（与 Projects 页 selectProject 一致）
  async function enterProject() {
    const name = result.project_name;
    if (!name) return;
    try { await api('POST', '/api/projects/select', { name }); } catch (e) {}
    currentProject.set(name);
    try { progress.set(await api('GET', '/api/progress')); } catch (e) {}
    try {
      const cfg = await api('GET', '/api/config');
      config.set(cfg);
      if (cfg && cfg.language) {
        projectLanguage.set(cfg.language);
        applyDefaultLocale(cfg.language);
      }
    } catch (e) {}
    try { settings.set(await api('GET', '/api/settings')); } catch (e) {}
    try { chatSessions.set(await api('GET', '/api/chat/sessions')); } catch (e) {}
    currentChatSession.set(null);
    hide();
    navigate('config');
  }
</script>

{#if visible}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="fixed inset-0 z-[120] bg-black/60 backdrop-blur-sm flex items-center justify-center" on:click={hide}>
    <div class="bg-base-200 rounded-xl shadow-2xl p-6 w-[92%] max-w-md max-h-[85vh] overflow-y-auto border border-base-content/10" on:click|stopPropagation>
      <div class="flex items-center gap-2 mb-4">
        <span class="text-xl">🌱</span>
        <h3 class="text-lg font-semibold">从世界播种小说</h3>
        <div class="flex-1"></div>
        <button class="btn btn-ghost btn-sm btn-circle" on:click={hide}>✕</button>
      </div>

      {#if !result}
        <label class="label py-0 text-xs text-base-content/60">选择世界（world_id）</label>
        {#if worldList.length}
          <select class="select select-sm select-bordered w-full mb-2" bind:value={selectedWorld}>
            <option value="">（手动输入 world_id）</option>
            {#each worldList as w (w.name)}
              <option value={w.name}>{w.name}{w.day > 0 ? ' · D' + w.day : ''}{w.active ? '（当前）' : ''}</option>
            {/each}
          </select>
        {/if}
        <input
          class="input input-sm input-bordered w-full mb-2"
          placeholder="世界ID（如：临江市·都市怪谈 或 九州·凡尘仙途）"
          bind:value={manualWorld}
          disabled={worldList.length > 0 && selectedWorld}
        />
        {#if !worldList.length}
          <p class="text-xs text-base-content/40 mb-2">⚠️ 未获取到世界列表，请手动填写已模拟世界名。</p>
        {/if}

        <label class="label py-0 text-xs text-base-content/60">项目名（storys/ 下新建）</label>
        <input class="input input-sm input-bordered w-full mb-2" placeholder="如：临江异闻录·小说版" bind:value={projectName} />

        <label class="label py-0 text-xs text-base-content/60">语言</label>
        <div class="join mb-2">
          <button type="button" class="btn btn-sm join-item {language === 'zh' ? 'btn-primary' : 'btn-ghost'}" on:click={() => language = 'zh'}>中文</button>
          <button type="button" class="btn btn-sm join-item {language === 'en' ? 'btn-primary' : 'btn-ghost'}" on:click={() => language = 'en'}>EN</button>
        </div>

        {#if err}
          <div class="alert alert-error text-sm py-2 mb-2">⚠️ {err}</div>
        {/if}

        <div class="flex gap-2 mt-3">
          <button class="btn btn-primary btn-sm flex-1" on:click={seed} disabled={seeding || !worldId || !projectName.trim()}>
            {#if seeding}
              <span class="loading loading-spinner loading-xs"></span> 播种中…
            {:else}
              🌱 播种并创建项目
            {/if}
          </button>
          <button class="btn btn-ghost btn-sm" on:click={hide}>取消</button>
        </div>
      {:else}
        <div class="rounded-lg border border-primary/30 bg-primary/10 p-4 space-y-1.5 text-sm">
          <div class="font-semibold text-primary mb-1">播种完成：{result.project_name}</div>
          <div class="flex justify-between"><span class="text-base-content/60">源世界</span><span>{result.world_name || worldId}</span></div>
          <div class="flex justify-between"><span class="text-base-content/60">语言</span><span>{result.language === 'en' ? 'EN' : '中文'}</span></div>
          <div class="flex justify-between"><span class="text-base-content/60">角色数</span><span>{result.character_count ?? '-'}</span></div>
          <div class="flex justify-between"><span class="text-base-content/60">世界观数</span><span>{result.worldview_count ?? '-'}</span></div>
          <div class="flex justify-between"><span class="text-base-content/60">组织数</span><span>{result.organization_count ?? '-'}</span></div>
          <div class="flex justify-between"><span class="text-base-content/60">大纲章节</span><span>{result.outline_chapters ?? '-'}</span></div>
          <div class="flex justify-between"><span class="text-base-content/60">模拟天数</span><span>{result.day_count ?? '-'}</span></div>
          {#if result.reused}
            <div class="text-xs text-base-content/50 pt-1">（复用了世界角色/世界观设定作为种子）</div>
          {/if}
        </div>
        <div class="flex gap-2 mt-4">
          <button class="btn btn-primary btn-sm flex-1" on:click={enterProject}>✍️ 进入该小说项目</button>
          <button class="btn btn-ghost btn-sm" on:click={hide}>关闭</button>
        </div>
      {/if}
    </div>
  </div>
{/if}