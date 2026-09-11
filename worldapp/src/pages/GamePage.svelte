<script>
  import { openTab, navigateTo } from '../lib/browser.js';

  // 数字游戏游玩页：网关旁的完整地址（在网关页内嵌终端风游戏面板）
  const gameApi = (p, opt) => fetch(p, { headers: { 'Content-Type': 'application/json' }, ...opt }).then((r) => r.json());

  let state = null;
  let busy = false;
  let input = '';
  let mode = 'do';
  let turns = [];
  let started = false;

  async function load() {
    try {
      const r = await gameApi('/api/game/status');
      if (r && r.game) {
        state = r.game;
        turns = (state.log || []).map((e) => ({
          who: e.mode === 'start' ? '开场' : `你(${e.mode})`,
          text: e.mode === 'start' ? e.narration : e.input,
          reply: e.mode === 'start' ? '' : e.narration,
          check: e.check,
        }));
        started = !!state.enabled;
      }
    } catch (e) {
      turns = [{ who: '提示', text: '无法连接世界模拟服务（/api/game/*），进程可能未启动', reply: '', check: null }];
    }
  }
  load();

  async function send() {
    const txt = input.trim();
    if (!txt || busy) return;
    input = '';
    busy = true;
    turns = [...turns, { who: `你(${modeName()})`, text: txt, reply: '', check: null }];
    try {
      const r = await gameApi('/api/game/action', {
        method: 'POST',
        body: JSON.stringify({ input: txt, mode }),
      });
      if (r.ok) {
        state = r.state;
        const last = (state.log || []).slice(-1)[0];
        turns = [...turns, { who: '世界', text: r.narration, reply: '', check: last && last.check }];
      } else {
        turns = [...turns, { who: '系统', text: r.error || '未知错误', reply: '', check: null }];
      }
    } catch (e) {
      turns = [...turns, { who: '系统', text: '请求失败：' + e, reply: '', check: null }];
    }
    busy = false;
  }

  async function wait() {
    if (busy) return;
    busy = true;
    try {
      const r = await gameApi('/api/game/wait', { method: 'POST' });
      if (r.ok) {
        state = r.state;
        turns = [...turns, { who: '等待中', text: '（观察世界继续运转）', reply: r.narration, check: null }];
      } else turns = [...turns, { who: '系统', text: r.error, reply: '', check: null }];
    } catch (e) {
      turns = [...turns, { who: '系统', text: '' + e, reply: '', check: null }];
    }
    busy = false;
  }

  async function startGame() {
    if (busy) return;
    busy = true;
    try {
      const r = await gameApi('/api/game/start', { method: 'POST', body: JSON.stringify({ input: '', mode }) });
      if (r.ok) {
        state = r.state;
        started = true;
        turns = ((state.log || []).filter((e) => e.mode === 'start')).map((e) => ({ who: '开场', text: e.narration, reply: '', check: null }));
      } else turns = [...turns, { who: '系统', text: r.error, reply: '', check: null }];
    } catch (e) {
      turns = [...turns, { who: '系统', text: '' + e, reply: '', check: null }];
    }
    busy = false;
  }

  async function stopGame() {
    try {
      await gameApi('/api/game/stop', { method: 'POST' });
    } catch (_e) { /* 忽略 */ }
    started = false;
  }

  const modeName = () => ({ do: '行动', say: '说话', story: '叙事' }[mode] || '行动');
</script>

<div class="page-enter max-w-5xl mx-auto p-6">
  <div class="paper card card-body p-6">
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div>
        <h2 class="text-2xl font-bold flex items-center gap-2">🎲 文字游戏</h2>
        <p class="text-base-content/60 mt-1 text-sm">同一世界，亲手下场。输入自由文本，命运在骰子上——代码掷骰 d20 vs 难度，数值由代码持有，AI 只裁决与叙述。</p>
      </div>
      <div class="flex gap-2">
        {#if !started}
          <button class="btn btn-primary btn-sm" on:click={startGame} disabled={busy}>▶ 开始 / 重开</button>
        {:else}
          <button class="btn btn-ghost btn-sm" on:click={stopGame}>■ 结束</button>
        {/if}
      </div>
    </div>

    {#if started && state}
      <!-- 面板 -->
      <div class="flex flex-wrap gap-2 mt-4 text-sm">
        <span class="badge bg-primary/10 text-primary">LV {state.level}</span>
        <span class="badge bg-success/10 text-success">HP {state.hp}/{state.max_hp}</span>
        <span class="badge bg-warning/10 text-warning">💰 {state.gold}</span>
        <span class="badge bg-accent/10 text-accent">⭐XP {state.xp}/100</span>
        {#each Object.entries(state.attrs || {}) as [k, v] ([k])}
          <span class="badge badge-ghost">{k} {v}/10</span>
        {/each}
      </div>
      <div class="flex flex-wrap gap-2 mt-2 text-xs text-base-content/60">
        📍 {state.location || '—'} ｜ 🎯 {state.quest || '—'} ｜ 🎒 {(state.inventory || []).length ? state.inventory.join('、') : '（空）'}
      </div>

      <!-- 账本 -->
      <div class="mt-4 space-y-3 max-h-[46vh] overflow-y-auto pr-1">
        {#each turns as t, i (i)}
          <div class="border-l-2 border-primary/60 pl-3">
            <div class="text-xs opacity-60">{t.who}</div>
            <div class="text-sm whitespace-pre-wrap leading-relaxed">{t.text}</div>
            {#if t.reply}<div class="text-sm whitespace-pre-wrap leading-relaxed mt-1">{t.reply}</div>{/if}
            {#if t.check}
              <div class="text-xs mt-1 {t.check.success ? 'text-success' : 'text-error'}">
                🎲 检定[{t.check.ability}]: d20={t.check.roll}{t.check.mod >= 0 ? '+' : ''}{t.check.mod} vs 难度{t.check.dc} → {t.check.success ? '成功' : '失败'}
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <!-- 输入区 -->
      <div class="mt-4">
        <div class="flex gap-2 mb-2">
          {#each [['do', '▸ 行动'], ['say', '💬 说话'], ['story', '✍ 叙事']] as [m, label] ([m])}
            <button class="btn btn-ink btn-sm {mode === m ? 'btn-primary' : ''}" on:click={() => (mode = m)}>{label}</button>
          {/each}
          <button class="btn btn-ghost btn-sm" on:click={wait} disabled={busy}>⏳ 等待（世界自转+回血）</button>
        </div>
        <div class="flex gap-2">
          <input class="input input-bordered input-sm flex-1" placeholder="输入你要做的事，回车提交…" bind:value={input} on:keydown={(e) => e.key === 'Enter' && send()} disabled={busy} />
          <button class="btn btn-primary btn-sm" on:click={send} disabled={busy}>⏎</button>
        </div>
        {#if busy}<p class="text-xs text-primary mt-2">裁决中……（掷骰 + 叙述需要两次 LLM 调用）</p>{/if}
      </div>
    {:else}
      <div class="mt-6 text-center text-base-content/60 py-10 border border-dashed border-base-content/20 rounded-box">
        <p>尚无进行中的游戏局。</p>
        <p class="text-xs mt-2">先到「世界模拟」建世界并初始化主角，然后点上方「开始」进入游玩。回合制：等待回合世界会自己往前走。</p>
      </div>
    {/if}
  </div>
</div>
