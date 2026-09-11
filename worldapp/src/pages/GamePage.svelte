<script>
  import { toast } from '../apps/world/lib/stores.js';

  // 文字游戏游玩页：自由输入 + 代码掷骰 + AI 叙事；数值/背包/好感/撤销/存档全部由代码层持有。
  const gameApi = (p, opt) => fetch(p, { headers: { 'Content-Type': 'application/json' }, ...opt }).then((r) => r.json().catch(() => ({ ok: false, error: '非 JSON 响应' })));

  let state = null;
  let busy = false;
  let input = '';
  let mode = 'do';
  let turns = [];
  let started = false;
  let heroSrc = '';
  let sceneSrc = '';
  let portraits = [];
  let itemIcons = [];
  let mapData = null;
  let worldName = '';
  let worldDay = null;
  let wiHits = [];
  let hist = [];
  let histIdx = -1;
  let importInput;

  function applyStatus(r) {
    if (!r || !r.game) return;
    state = r.game;
    started = !!state.enabled;
    heroSrc = r.pixel?.hero || '';
    sceneSrc = r.scene || '';
    portraits = r.portraits || [];
    itemIcons = r.item_icons || [];
    mapData = r.map || null;
    worldName = r.world || worldName;
    worldDay = r.day ?? worldDay;
    wiHits = r.wi_hits || [];
    turns = (state.log || []).map((e) => ({
      who: e.mode === 'start' ? '开场' : `你(${modeLabel(e.mode)})`,
      text: e.mode === 'start' ? e.narration : e.input,
      reply: e.mode === 'start' ? '' : e.narration,
      check: e.check,
    }));
  }

  async function load() {
    try {
      applyStatus(await gameApi('/api/game/status'));
    } catch (e) {
      turns = [{ who: '提示', text: '无法连接世界模拟服务（/api/game/*），进程可能未启动', reply: '', check: null }];
    }
  }
  load();

  async function send() {
    const txt = input.trim();
    if (!txt || busy) return;
    input = '';
    histIdx = -1;
    busy = true;
    hist.push(txt);
    if (hist.length > 30) hist.shift();
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
        await refreshView();
      } else {
        turns = [...turns, { who: '系统', text: r.error || '未知错误', reply: '', check: null }];
      }
    } catch (e) {
      turns = [...turns, { who: '系统', text: '请求失败：' + e, reply: '', check: null }];
    }
    busy = false;
  }

  async function refreshView() {
    try {
      const fresh = await gameApi('/api/game/status');
      if (fresh && fresh.ok) {
        const keep = turns;
        applyStatus(fresh);
        turns = keep; // 新回合已本地追加，全量刷新只更新观感字段
      }
    } catch (_e) { /* 刷新失败不阻塞回合 */ }
  }

  async function wait() {
    if (busy) return;
    busy = true;
    turns = [...turns, { who: '等待', text: '（观察世界继续运转）', reply: '', check: null }];
    try {
      const r = await gameApi('/api/game/wait', { method: 'POST' });
      if (r.ok) {
        turns = [...turns.slice(0, -1), { who: '等待', text: '（观察世界继续运转）', reply: r.narration, check: null }];
        state = r.state;
        await refreshView();
      } else turns = [...turns.slice(0, -1), { who: '系统', text: r.error, reply: '', check: null }];
    } catch (e) {
      turns = [...turns.slice(0, -1), { who: '系统', text: '' + e, reply: '', check: null }];
    }
    busy = false;
  }

  async function undo() {
    if (busy) return;
    busy = true;
    try {
      const r = await gameApi('/api/game/undo', { method: 'POST' });
      if (r.ok) {
        toast('已撤销上一步', 'success');
        await load();
      } else toast(r.error || '撤销失败', 'error');
    } catch (e) {
      toast('撤销失败：' + e, 'error');
    }
    busy = false;
  }

  function exportSave() {
    window.location.href = '/api/game/export';
  }

  async function importSave(e) {
    const f = e.target.files?.[0];
    if (!f) return;
    busy = true;
    try {
      const data = JSON.parse(await f.text());
      const r = await gameApi('/api/game/import', { method: 'POST', body: JSON.stringify(data) });
      if (r.ok) {
        toast('存档已导入', 'success');
        await load();
      } else toast(r.error || '导入失败', 'error');
    } catch (err) {
      toast('读档失败：' + err, 'error');
    }
    busy = false;
    e.target.value = '';
  }

  async function startGame() {
    if (started && !confirm('重开将清空当前进度（可先「存档」备份），确定？')) return;
    if (busy) return;
    busy = true;
    try {
      const r = await gameApi('/api/game/start', { method: 'POST' });
      if (r.ok) {
        await load();
        const box = document.querySelector('.ledger');
        if (box) box.scrollTop = 0;
      } else toast(r.error, 'error');
    } catch (e) {
      toast('' + e, 'error');
    }
    busy = false;
  }

  async function stopGame() {
    try {
      await gameApi('/api/game/stop', { method: 'POST' });
    } catch (_e) { /* 忽略 */ }
    await load();
  }

  function onKeydown(e) {
    if (e.key === 'Enter') { send(); return; }
    if (e.key === 'ArrowUp') {
      if (histIdx < hist.length - 1) { histIdx++; input = hist[hist.length - 1 - histIdx] || ''; }
      e.preventDefault();
    } else if (e.key === 'ArrowDown') {
      if (histIdx > 0) { histIdx--; input = hist[hist.length - 1 - histIdx] || ''; }
      else { histIdx = -1; input = ''; }
      e.preventDefault();
    }
  }

  const modeName = () => ({ do: '行动', say: '说话', story: '叙事', wait: '等待' }[mode] || '行动');
  const modeLabel = (m) => ({ do: '行动', say: '说话', story: '叙事', wait: '等待' }[m] || m);
  const hpPct = (s) => Math.max(0, Math.min(100, (100 * (s?.hp ?? 0)) / Math.max(1, s?.max_hp ?? 1)));
</script>

<div class="page-enter max-w-5xl mx-auto p-6">
  <div class="paper card card-body p-6">
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div>
        <h2 class="text-2xl font-bold flex items-center gap-2">🎲 文字游戏</h2>
        <p class="text-base-content/60 mt-1 text-sm">
          {#if worldName}世界：{worldName}{#if worldDay != null} · Day {worldDay}{/if} ｜ {/if}输入自由文本，命运在骰子上——代码掷骰 d20 vs 难度，数值由代码持有，AI 只裁决与叙述。
        </p>
      </div>
      <div class="flex gap-2">
        {#if !started}
          <button class="btn btn-primary btn-sm" on:click={startGame} disabled={busy}>▶ 开始</button>
        {:else}
          <button class="btn btn-ghost btn-sm" on:click={() => startGame()} disabled={busy}>↻ 重开</button>
          <button class="btn btn-ghost btn-sm" on:click={stopGame} disabled={busy}>■ 结束</button>
        {/if}
        <button class="btn btn-ghost btn-sm" on:click={exportSave} title="下载 game.json 存档">💾</button>
        <button class="btn btn-ghost btn-sm" on:click={() => importInput.click()} title="导入 game.json 存档">📥</button>
        <input type="file" accept=".json,application/json" class="hidden" bind:this={importInput} on:change={importSave} />
      </div>
    </div>

    {#if started && state}
      <!-- 倒下横幅 -->
      {#if state.downed}
        <div class="mt-4 alert alert-error text-sm py-2">
          <span>⚠ 你倒下了（HP 0）——用「等待」回合回血恢复意识，或重开。</span>
        </div>
      {/if}

      <!-- WI 动态情报透明化 -->
      {#if wiHits.length}
        <div class="mt-3 flex flex-wrap gap-1.5 text-[11px]">
          {#each wiHits as k (k)}
            <span class="badge badge-sm border-violet-500/40 bg-violet-500/10 text-violet-300">🔎 {k}</span>
          {/each}
        </div>
      {/if}

      <!-- 场景横幅（晨/暮/夜按世界时钟轮转） -->
      {#if sceneSrc}
        <img src={sceneSrc} alt="场景" class="mt-4 w-full rounded-lg border border-base-content/10" style="height:120px;object-fit:cover;object-position:center 30%;image-rendering:pixelated" onerror={() => (sceneSrc = '')} />
      {/if}

      <!-- 面板 -->
      <div class="flex flex-wrap gap-2 mt-4 text-sm items-center">
        <span class="badge bg-primary/10 text-primary">LV {state.level}</span>
        <span class="badge {hpPct(state) <= 25 ? 'bg-error/15 text-error' : hpPct(state) <= 55 ? 'bg-warning/15 text-warning' : 'bg-success/10 text-success'}">
          HP {state.hp}/{state.max_hp}
        </span>
        <div class="w-24 h-1.5 rounded bg-base-content/10 overflow-hidden" title="HP {state.hp}/{state.max_hp}">
          <div class="h-full {hpPct(state) <= 25 ? 'bg-error' : hpPct(state) <= 55 ? 'bg-warning' : 'bg-success'}" style="width:{hpPct(state)}%;transition:width .3s"></div>
        </div>
        <span class="badge bg-warning/10 text-warning">💰 {state.gold}</span>
        <span class="badge bg-accent/10 text-accent">⭐XP {state.xp}/100</span>
        {#each Object.entries(state.attrs || {}) as [k, v] ([k])}
          <span class="badge badge-ghost">{k} {v}/10</span>
        {/each}
        {#each Object.entries(state.relations || {}) as [n, r] ([n])}
          <span class="badge badge-ghost {r > 0 ? 'text-success' : r < 0 ? 'text-error' : 'opacity-60'}">❤ {n} {r > 0 ? '+' : ''}{r}</span>
        {/each}
      </div>
      <div class="flex flex-wrap gap-2 mt-2 text-xs text-base-content/60">
        📍 {state.location || '—'} ｜ 🎯 {state.quest || '—'} ｜ 🎒 {(state.inventory || []).length ? state.inventory.join('、') : '（空）'}
      </div>

      <!-- 像素主角 + 在场角色头像（美术工坊素材） -->
      {#if heroSrc || portraits.length}
        <div class="mt-3 flex items-end gap-3 flex-wrap">
          {#if heroSrc}<img src={heroSrc} alt="主角" style="height:96px;image-rendering:pixelated" onerror={() => (heroSrc = '')} />{/if}
          {#each portraits as p (p.name)}
            <div class="text-center">
              {#if p.img}<img src={p.img} alt={p.name} style="height:56px;image-rendering:pixelated" class="rounded-full" onerror={() => (p.img = '')} />{/if}
              <div class="text-xs mt-1">{p.name}{#if p.relation !== undefined}<span class="ml-1 {p.relation > 0 ? 'text-success' : p.relation < 0 ? 'text-error' : 'opacity-50'}">{p.relation > 0 ? '+' : ''}{p.relation}</span>{/if}</div>
            </div>
          {/each}
        </div>
      {/if}

      <!-- 背包（物品图 + 名称） -->
      {#if (itemIcons && itemIcons.length) || (state.inventory || []).length}
        <div class="mt-2 flex flex-wrap gap-1.5">
          {#each (itemIcons.length ? itemIcons : (state.inventory || []).map((n) => ({ name: n }))) as it, i (i)}
            <span class="badge badge-ghost gap-1 py-1 pr-2">
              {#if it.img}<img src={it.img} alt="" style="width:22px;height:22px;image-rendering:pixelated;border-radius:4px" onerror={() => (it.img = '')} />{/if}
              {it.name}
            </span>
          {/each}
        </div>
      {/if}

      <!-- 确定性地图（tile 渲染，含在场 NPC 计数） -->
      {#if mapData && mapData.cells && mapData.cells.length}
        <details class="mt-3">
          <summary class="text-xs opacity-60 cursor-pointer">🗺 世界地图（确定性布局 · tile 渲染）</summary>
          <div class="mt-2 grid gap-0.5 w-fit" style="grid-template-columns: repeat({mapData.side}, 64px)">
            {#each mapData.cells as c (c.name)}
              <div
                class="relative rounded {c.x === mapData.hero?.x && c.y === mapData.hero?.y ? 'outline outline-2 outline-primary' : 'outline outline-1 outline-base-content/10'}"
                style="width:64px;height:64px;background-image:url('{(mapData.base || '/art/') || ''}tile-{c.tile}.png');background-size:cover;image-rendering:pixelated;background-color:#181f27"
                title={c.name}
              >
                {#if c.x === mapData.hero?.x && c.y === mapData.hero?.y}
                  <span class="absolute inset-0 flex items-center justify-center text-base">🧍</span>
                  {#if mapData.npcs && mapData.npcs.length}
                    <span class="absolute right-1 top-0.5 text-[10px] opacity-80">👥{mapData.npcs.length}</span>
                  {/if}
                {/if}
              </div>
            {/each}
          </div>
        </details>
      {/if}

      <!-- 账本 -->
      <div class="ledger mt-4 space-y-3 max-h-[46vh] overflow-y-auto pr-1">
        {#each turns as t, i (i)}
          <div class="border-l-2 border-primary/60 pl-3">
            <div class="text-xs opacity-60">{t.who}</div>
            <div class="text-sm whitespace-pre-wrap leading-relaxed">{t.text}</div>
            {#if t.reply}<div class="text-sm whitespace-pre-wrap leading-relaxed mt-1">{t.reply}</div>{/if}
            {#if t.check}
              <div class="text-xs mt-1 {t.check.success ? 'text-success' : 'text-error'} dice-line">
                🎲 检定[{t.check.ability}]: d20={t.check.roll}{t.check.mod >= 0 ? '+' : ''}{t.check.mod} vs 难度{t.check.dc} → {t.check.success ? '成功' : '失败'}
              </div>
            {/if}
          </div>
        {/each}
      </div>

      <!-- 输入区 -->
      <div class="mt-4">
        <div class="flex gap-2 mb-2 flex-wrap">
          {#each [['do', '▸ 行动'], ['say', '💬 说话'], ['story', '✍ 叙事']] as [m, label] ([m])}
            <button class="btn btn-ink btn-sm {mode === m ? 'btn-primary' : ''}" on:click={() => (mode = m)}>{label}</button>
          {/each}
          <button class="btn btn-ghost btn-sm" on:click={wait} disabled={busy}>⏳ 等待（自转+回血）</button>
          <button class="btn btn-ghost btn-sm" on:click={undo} disabled={busy} title="数值/背包/好感回滚到上一回合前">↩ 撤销上一步</button>
        </div>
        <div class="flex gap-2">
          <input class="input input-bordered input-sm flex-1" placeholder="输入你要做的事，↑/↓ 翻历史，回车提交…" bind:value={input} on:keydown={onKeydown} disabled={busy} />
          <button class="btn btn-primary btn-sm" on:click={send} disabled={busy}>⏎</button>
        </div>
        {#if busy}<p class="text-xs text-primary mt-2">裁决中……（掷骰 + 叙述需要两次 LLM 调用）</p>{/if}
      </div>
    {:else}
      <div class="mt-6 text-center text-base-content/60 py-10 border border-dashed border-base-content/20 rounded-box">
        <p>尚无进行中的游戏局。</p>
        <p class="text-xs mt-2">先到「世界模拟」建世界并初始化主角，然后点上方「开始」进入游玩。回合制：等待回合世界会自己往前走（引擎日推进一天）。</p>
      </div>
    {/if}
  </div>
</div>

<style>
  @keyframes rollIn {
    0% { transform: translateY(-14px) rotate(-40deg); opacity: 0; }
    60% { transform: translateY(3px) rotate(8deg); opacity: 1; }
    100% { transform: none; }
  }
  .dice-line { animation: rollIn 0.5s ease-out; }
</style>
