<script>
  import { currentPage, navigate } from './lib/router.js';
  import { progress, taskRunning, contextPage, toastStore, currentProject, projectLanguage } from './lib/stores.js';
  import { connectSSE } from './lib/sse.js';
  import { api } from './lib/api.js';
  import { onMount } from 'svelte';
  import { t, uiLocale, setLocale } from './lib/i18n/index.js';
  import TaskTokenBadge from './components/TaskTokenBadge.svelte';
  import Projects from './pages/Projects.svelte';
  import Config from './pages/Config.svelte';
  import Outline from './pages/Outline.svelte';
  import Writing from './pages/Writing.svelte';
  import Relations from './pages/Relations.svelte';
  import Skills from './pages/Skills.svelte';
  import Foreshadows from './pages/Foreshadows.svelte';
  import Memory from './pages/Memory.svelte';
  import ChatPanel from './components/ChatPanel.svelte';
  import ConfirmModal from './components/ConfirmModal.svelte';

  let chatPanel;

  $: $contextPage = $currentPage;

  onMount(async () => {
    connectSSE();
    try {
      const cur = await api('GET', '/api/projects/current');
      if (cur.name) {
        currentProject.set(cur.name);
        if (cur.language) {
          projectLanguage.set(cur.language);
          setLocale(cur.language);
        }
        try { const p = await api('GET', '/api/progress'); progress.set(p); } catch (e) {}
      }
    } catch (e) {}
  });

  async function sendToChat(text) {
    if (chatPanel) await chatPanel.sendMessageToChat(text);
  }

  function backToProjects() {
    currentProject.set(null);
  }

  function toggleLocale() {
    setLocale($uiLocale === 'en' ? 'zh' : 'en');
  }
</script>

<div class="flex flex-col h-full bg-base-300 text-base-content overflow-hidden">
  <!-- 精简顶栏（放入浏览器外壳内，不再重复品牌/版本） -->
  <div class="navbar bg-base-200 border-b border-base-content/10 px-4 min-h-[40px] shrink-0 gap-3">
    {#if $currentProject}
      <span class="badge badge-sm badge-outline">{$currentProject}</span>
      <span class="badge badge-sm badge-accent uppercase" title={$projectLanguage === 'en' ? 'English' : '中文'}>
        {$projectLanguage === 'en' ? 'EN' : 'ZH'}
      </span>
      <button class="btn btn-ghost btn-xs gap-1" on:click={backToProjects} disabled={$taskRunning}>
        {$t('app.switchProject')}
      </button>
      <span class="badge badge-sm" class:badge-primary={$progress}>
        {$progress ? ($progress.phase === 'outline' ? $t('app.phase.outline') : $progress.phase === 'writing' ? $t('app.phase.writing') : $progress.phase) : $t('app.phase.unstarted')}
      </span>
      {#if $taskRunning}
        <span class="badge badge-sm badge-warning gap-1">
          <span class="loading loading-spinner loading-xs"></span>
          {$t('app.aiThinking')}
          <TaskTokenBadge className="badge badge-xs badge-warning font-mono border-0" />
        </span>
      {/if}
    {/if}
    <span class="flex-1"></span>
    <button class="btn btn-ghost btn-xs gap-1" on:click={toggleLocale} title={$t('app.uiLang.label')}>
      {$uiLocale === 'en' ? $t('app.uiLang.en') : $t('app.uiLang.zh')}
    </button>
  </div>

  {#if !$currentProject}
    <main class="flex-1 overflow-y-auto p-6"><Projects /></main>
  {:else}
    <div class="flex flex-1 overflow-hidden">
      <nav class="flex flex-col w-44 shrink-0 bg-base-200 border-r border-base-content/10 py-3 px-2 gap-0.5">
        {#each [
          ['config', '⚙️', 'nav.config'],
          ['outline', '📝', 'nav.outline'],
          ['writing', '✍️', 'nav.writing'],
          ['foreshadows', '🔗', 'nav.foreshadows'],
          ['memory', '🧠', 'nav.memory'],
          ['relations', '🕸️', 'nav.relations'],
          ['skills', '🧩', 'nav.skills']
        ] as [page, icon, labelKey]}
          <button
            class="btn btn-sm justify-start w-full gap-2 px-3 text-sm {$currentPage === page ? 'btn-primary font-medium' : 'btn-ghost'}"
            on:click={() => navigate(page)}
          >
            <span class="text-xs">{icon}</span>{$t(labelKey)}
          </button>
        {/each}
      </nav>

      <main class="flex-1 min-w-0 overflow-y-auto p-4 border-r border-base-content/10">
        {#if $currentPage === 'config'}
          <Config {sendToChat} />
        {:else if $currentPage === 'outline'}
          <Outline {sendToChat} />
        {:else if $currentPage === 'writing'}
          <Writing {sendToChat} />
        {:else if $currentPage === 'foreshadows'}
          <Foreshadows />
        {:else if $currentPage === 'memory'}
          <Memory />
        {:else if $currentPage === 'relations'}
          <Relations />
        {:else if $currentPage === 'skills'}
          <Skills />
        {/if}
      </main>

      <div class="flex-1 min-w-0 bg-base-200 overflow-hidden">
        <ChatPanel bind:this={chatPanel} contextPage={$currentPage} />
      </div>
    </div>
  {/if}

  <div class="fixed top-5 right-5 z-50 flex flex-col gap-2">
    {#each $toastStore as t (t.id)}
      <div class="alert alert-sm {t.type === 'success' ? 'alert-success' : t.type === 'error' ? 'alert-error' : 'alert-info'} toast-enter shadow-lg max-w-sm">
        <span>{t.msg}</span>
      </div>
    {/each}
  </div>

  <ConfirmModal />
</div>