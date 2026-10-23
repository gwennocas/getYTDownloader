<script>
  import { onMount } from 'svelte';
  import {
    ArrowDownToLine, ArrowRight, ArrowUpRight, Check, CheckCircle2,
    ChevronDown, CircleAlert, Clipboard, Download, Film, FolderOpen,
    Headphones, Info, Link2, ListVideo, LoaderCircle, Music2, Play,
    RefreshCw, Search, Settings2, ShieldCheck, Sparkles, Square,
    WandSparkles, X
  } from '@lucide/svelte';

  let page = 'download';
  let url = '';
  let folder = '';
  let mode = 'video';
  let quality = 'best';
  let playlist = false;
  let preview = null;
  let inspecting = false;
  let running = false;
  let lastEvent = null;
  let logs = [];
  let currentItem = '';
  let error = '';
  let notice = '';
  let updatingYt = false;
  let updatingFFmpeg = false;
  let checkingApp = false;
  let installingApp = false;
  let update = null;
  let state = { version: 'dev', ytdlpReady: false, ytdlpVersion: '', ffmpegReady: false, downloadPath: '', active: false };

  const bridge = () => window.go?.main?.App;
  async function invoke(name, ...args) {
    const method = bridge()?.[name];
    if (!method) throw new Error('Abra o aplicativo pelo Wails para usar esta função.');
    return method(...args);
  }

  function message(err) {
    return typeof err === 'string' ? err : (err?.message || String(err));
  }

  function clearAlerts() { error = ''; notice = ''; }

  async function refreshState() {
    try {
      state = await invoke('GetState');
      folder = localStorage.getItem('getyt-folder') || state.downloadPath;
      running = state.active;
    } catch (err) { error = message(err); }
  }

  onMount(() => {
    refreshState();
    const off = window.runtime?.EventsOn?.('download', event => {
      lastEvent = { ...lastEvent, ...event };
      if (event.status === 'running' || event.status === 'progress' || event.status === 'item') running = true;
      if (event.status === 'item') currentItem = event.message;
      if (event.status === 'log') logs = [...logs.slice(-49), event.message];
      if (event.status === 'error') { error = event.message; running = false; }
      if (event.status === 'cancelled') { notice = event.message; running = false; }
      if (event.status === 'done') { notice = event.message; running = false; }
    });
    return () => off?.();
  });

  async function paste() {
    clearAlerts();
    try { url = await navigator.clipboard.readText(); preview = null; }
    catch { error = 'Não foi possível ler a área de transferência. Cole o link manualmente.'; }
  }

  async function inspect() {
    clearAlerts();
    if (!url.trim()) { error = 'Cole o link de um vídeo ou playlist.'; return; }
    inspecting = true;
    try {
      preview = await invoke('Inspect', url.trim());
      if (preview.isPlaylist) playlist = true;
    } catch (err) { preview = null; error = message(err); }
    finally { inspecting = false; }
  }

  async function chooseFolder() {
    clearAlerts();
    try {
      const selected = await invoke('ChooseFolder');
      if (selected) { folder = selected; localStorage.setItem('getyt-folder', selected); }
    } catch (err) { error = message(err); }
  }

  async function start() {
    clearAlerts();
    if (!url.trim()) { error = 'Cole o link antes de começar.'; return; }
    if (!folder) { error = 'Escolha a pasta de destino.'; return; }
    try {
      await invoke('StartDownload', { url: url.trim(), folder, mode, quality, playlist });
      running = true;
      logs = [];
      currentItem = preview?.title || 'Preparando mídia';
      lastEvent = { status: 'running', message: 'Preparando download…', percent: '0%' };
      localStorage.setItem('getyt-folder', folder);
    } catch (err) { error = message(err); }
  }

  async function cancel() {
    try { await invoke('CancelDownload'); notice = 'Cancelando download…'; }
    catch (err) { error = message(err); }
  }

  async function openFolder() {
    try { await invoke('OpenFolder', folder); }
    catch (err) { error = message(err); }
  }

  async function updateYTDLP() {
    clearAlerts(); updatingYt = true;
    try {
      const v = await invoke('UpdateYTDLP');
      state = await invoke('GetState');
      notice = `yt-dlp ${v} instalado com sucesso.`;
    } catch (err) { error = message(err); }
    finally { updatingYt = false; }
  }

  async function updateFFmpeg() {
    clearAlerts(); updatingFFmpeg = true;
    try {
      notice = await invoke('UpdateFFmpeg');
      state = await invoke('GetState');
    } catch (err) { error = message(err); }
    finally { updatingFFmpeg = false; }
  }

  async function checkApp() {
    clearAlerts(); checkingApp = true;
    try {
      update = await invoke('CheckAppUpdate');
      if (!update.available) notice = 'Você já tem a versão mais recente.';
    } catch (err) { error = message(err); }
    finally { checkingApp = false; }
  }

  async function installApp() {
    clearAlerts(); installingApp = true;
    try { notice = await invoke('InstallAppUpdate'); }
    catch (err) { error = message(err); installingApp = false; }
  }

  function openRelease() {
    if (update?.url) window.runtime?.BrowserOpenURL?.(update.url);
  }

  $: progress = Math.max(0, Math.min(100, parseFloat(lastEvent?.percent || '0') || 0));
  $: statusLabel = running ? 'Baixando agora' : lastEvent?.status === 'done' ? 'Concluído' : lastEvent?.status === 'cancelled' ? 'Cancelado' : 'Pronto para começar';
  $: modeLabel = mode === 'video' ? 'Vídeo' : mode === 'mp3' ? 'Áudio MP3' : 'Áudio original';
</script>

<svelte:head>
  <title>GetYT Downloader — baixe com liberdade</title>
</svelte:head>

<div class="app-shell">
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-mark"><ArrowDownToLine size={22} strokeWidth={2.5} /></div>
      <div class="brand-copy"><strong>get<span>YT</span></strong><small>DOWNLOADER</small></div>
    </div>

    <div class="side-label">ESPAÇO DE TRABALHO</div>
    <nav aria-label="Navegação principal">
      <button class:active={page === 'download'} on:click={() => page = 'download'}><Download size={18} /> Baixar mídia <span class="nav-indicator"></span></button>
      <button class:active={page === 'updates'} on:click={() => page = 'updates'}><Settings2 size={18} /> Atualizações <span class="nav-indicator"></span></button>
    </nav>

    <div class="sidebar-bottom">
      <div class="side-tip"><Sparkles size={18} /><p><strong>Uma playlist inteira?</strong><br />Cole o link, ative a opção e deixe o resto com a gente.</p></div>
      <div class="side-version"><span class="status-dot"></span> getYT <span>{state.version}</span></div>
    </div>
  </aside>

  <main class="main-content">
    <div class="topbar">
      <div class="breadcrumb">Biblioteca <span>/</span> <strong>{page === 'download' ? 'Novo download' : 'Atualizações'}</strong></div>
      <div class="topbar-right"><span class="top-status"><span class:offline={!state.ytdlpReady} class="status-dot"></span>{state.ytdlpReady ? 'Motor pronto' : 'Instalar motor'}</span><span class="avatar">GY</span></div>
    </div>

    {#if error || notice}
      <div class:error-banner={!!error} class:notice-banner={!!notice} class="alert" role="status">
        {#if error}<CircleAlert size={18} />{error}{:else}<CheckCircle2 size={18} />{notice}{/if}
        <button aria-label="Fechar aviso" on:click={clearAlerts}><X size={16} /></button>
      </div>
    {/if}

    {#if page === 'download'}
      <div class="page-grid">
        <section class="primary-column">
          <div class="hero">
            <div class="eyebrow"><span class="eyebrow-line"></span> SEU CONTEÚDO, DO SEU JEITO</div>
            <h1>Baixe o que você <em>ama.</em></h1>
            <p>Vídeos, músicas e playlists completas em um só lugar. Simples, rápido e com controle de verdade.</p>
            <div class="hero-art" aria-hidden="true"><div class="disc disc-one"></div><div class="disc disc-two"></div><div class="disc disc-three"></div><div class="hero-icon"><Play size={35} fill="currentColor" /></div></div>
          </div>

          <div class="section-heading"><div><span class="step">01</span><h2>De onde vamos baixar?</h2></div><span>YOUTUBE E OUTROS SITES COMPATÍVEIS</span></div>
          <div class="card url-card">
            <label for="media-url">LINK DO VÍDEO OU PLAYLIST</label>
            <div class="url-field"><Link2 size={20} /><input id="media-url" type="url" bind:value={url} on:input={() => preview = null} on:keydown={e => e.key === 'Enter' && inspect()} placeholder="Cole aqui o endereço da mídia…" /><button title="Colar da área de transferência" on:click={paste}><Clipboard size={17} /><span>Colar</span></button></div>
            <div class="url-footer"><span><ShieldCheck size={15} /> Sem anúncios. Sem complicação.</span><button class="text-button" on:click={inspect} disabled={inspecting || !url.trim()}>{#if inspecting}<LoaderCircle size={16} class="spin" /> Analisando{:else}<Search size={16} /> Analisar link <ArrowRight size={15} />{/if}</button></div>
            {#if preview}
              <div class="preview">
                {#if preview.thumbnail}<img src={preview.thumbnail} alt="Miniatura da mídia" />{:else}<div class="preview-placeholder"><Film size={27} /></div>{/if}
                <div class="preview-info"><strong>{preview.title || 'Mídia encontrada'}</strong><span>{preview.uploader || 'Origem identificada'}{preview.isPlaylist ? ` · ${preview.entryCount || 'Vários'} itens` : ''}</span><small>{preview.isPlaylist ? 'PLAYLIST DETECTADA' : 'VÍDEO DETECTADO'}</small></div>
                <CheckCircle2 size={20} class="preview-check" />
              </div>
            {/if}
          </div>

          <div class="section-heading"><div><span class="step">02</span><h2>Como você prefere?</h2></div><span>ESCOLHA O FORMATO IDEAL</span></div>
          <div class="format-grid">
            <button class:selected={mode === 'video'} class="format-card" on:click={() => mode = 'video'}><span class="format-icon gold"><Film size={22} /></span><strong>Vídeo</strong><small>Imagem e som na melhor qualidade</small><span class="selection"><Check size={14} /></span></button>
            <button class:selected={mode === 'mp3'} class="format-card" on:click={() => mode = 'mp3'}><span class="format-icon blue"><Music2 size={22} /></span><strong>Áudio MP3</strong><small>Compatível com qualquer dispositivo</small><span class="selection"><Check size={14} /></span></button>
            <button class:selected={mode === 'original'} class="format-card" on:click={() => mode = 'original'}><span class="format-icon purple"><Headphones size={22} /></span><strong>Áudio original</strong><small>Sem conversão, com qualidade total</small><span class="selection"><Check size={14} /></span></button>
          </div>

          {#if mode === 'video'}
            <div class="quality-row"><div><strong>Qualidade do vídeo</strong><span>O melhor formato disponível dentro do limite escolhido</span></div><div class="select-wrap"><select bind:value={quality} aria-label="Qualidade do vídeo"><option value="best">Melhor disponível</option><option value="1080">Até 1080p</option><option value="720">Até 720p</option><option value="480">Até 480p</option></select><ChevronDown size={17} /></div></div>
          {/if}

          <div class="playlist-card"><div class="playlist-icon"><ListVideo size={22} /></div><div><strong>Baixar playlist inteira</strong><span>Organiza os arquivos em uma pasta e retoma itens já baixados.</span></div><label class="switch" aria-label="Baixar playlist inteira"><input type="checkbox" bind:checked={playlist} /><span class="slider"></span></label></div>

          <div class="section-heading destination-heading"><div><span class="step">03</span><h2>Onde salvar?</h2></div></div>
          <div class="folder-card"><div class="folder-icon"><FolderOpen size={22} /></div><div><small>PASTA DE DESTINO</small><strong title={folder}>{folder || 'Nenhuma pasta selecionada'}</strong></div><button class="secondary-button" on:click={chooseFolder}>Alterar pasta <ArrowRight size={15} /></button></div>

          <div class="action-row"><button class="download-button" on:click={start} disabled={running || !state.ytdlpReady}><ArrowDownToLine size={21} /> {running ? 'Download em andamento' : playlist ? 'Baixar playlist' : 'Iniciar download'} <ArrowRight size={18} /></button><span>{state.ytdlpReady ? 'Tudo pronto para começar' : 'Instale o yt-dlp em Atualizações'}</span></div>
        </section>

        <aside class="right-column">
          <div class="activity-card"><div class="panel-heading"><span class="panel-icon"><Download size={18} /></span><div><small>ACOMPANHE EM TEMPO REAL</small><h3>Seu download</h3></div></div>
            <div class="activity-visual"><div class="activity-ring" style={`--progress: ${progress}%`}><div><ArrowDownToLine size={27} /></div></div></div>
            <div class="activity-status"><span class:live={running} class="pill"><span class="status-dot"></span>{statusLabel}</span><h3>{running ? currentItem || 'Preparando mídia…' : lastEvent?.status === 'done' ? 'Tudo salvo com sucesso!' : 'Sua próxima mídia começa aqui'}</h3><p>{running ? `${modeLabel}${playlist ? ' · Playlist' : ''}` : 'Cole um link e escolha suas preferências para começar.'}</p></div>
            {#if running || lastEvent?.status === 'done'}<div class="progress-wrap"><div class="progress-meta"><span>Progresso</span><strong>{Math.round(progress)}%</strong></div><div class="progress-track"><span style={`width: ${progress}%`}></span></div>{#if running}<div class="progress-detail"><span>{lastEvent?.speed || 'Calculando velocidade'}</span><span>{lastEvent?.eta ? `ETA ${lastEvent.eta}` : ''}</span></div>{/if}</div>{/if}
            <div class="activity-actions">{#if running}<button class="cancel-button" on:click={cancel}><Square size={14} fill="currentColor" /> Cancelar</button>{:else}<button on:click={openFolder} disabled={!folder}><FolderOpen size={16} /> Abrir pasta <ArrowUpRight size={15} /></button>{/if}</div>
          </div>
          <div class="helper-card"><div class="helper-icon"><WandSparkles size={22} /></div><div><strong>Feito para ser simples.</strong><p>Seu arquivo fica na pasta escolhida. Se uma playlist for interrompida, é só iniciar de novo para continuar.</p></div></div>
          {#if logs.length > 0}<div class="log-card"><div class="log-heading"><span>ATIVIDADE RECENTE</span><span>{logs.length} eventos</span></div><div class="log-lines">{#each logs.slice(-5) as line}<p>{line}</p>{/each}</div></div>{/if}
        </aside>
      </div>
    {:else}
      <div class="updates-page"><div class="updates-hero"><div class="eyebrow"><span class="eyebrow-line"></span> CUIDANDO DO SEU APP</div><h1>Sempre na <em>melhor versão.</em></h1><p>Atualize os componentes que fazem a mágica acontecer. Tudo no seu tempo, com um clique.</p></div>
        <div class="update-grid">
          <div class="update-card"><div class="update-head"><div class="update-icon"><Download size={24} /></div><span class:ready={state.ytdlpReady} class="dependency-pill">{state.ytdlpReady ? 'INSTALADO' : 'PENDENTE'}</span></div><h2>yt-dlp</h2><p>O motor que encontra e baixa seus vídeos e playlists de centenas de sites.</p><div class="update-version"><span>Versão atual</span><strong>{state.ytdlpVersion || 'Não instalado'}</strong></div><button on:click={updateYTDLP} disabled={updatingYt || running}>{#if updatingYt}<LoaderCircle size={17} class="spin" /> Instalando…{:else}<RefreshCw size={17} /> {state.ytdlpReady ? 'Atualizar yt-dlp' : 'Instalar yt-dlp'}{/if}</button></div>
          <div class="update-card"><div class="update-head"><div class="update-icon"><Film size={24} /></div><span class:ready={state.ffmpegReady} class="dependency-pill">{state.ffmpegReady ? 'INSTALADO' : 'PENDENTE'}</span></div><h2>FFmpeg</h2><p>Necessário para converter MP3 e combinar vídeo e áudio em alta qualidade.</p><div class="update-version"><span>Status</span><strong>{state.ffmpegReady ? 'Pronto para uso' : 'Não instalado'}</strong></div><button on:click={updateFFmpeg} disabled={updatingFFmpeg || running}>{#if updatingFFmpeg}<LoaderCircle size={17} class="spin" /> Instalando…{:else}<RefreshCw size={17} /> {state.ffmpegReady ? 'Atualizar FFmpeg' : 'Instalar FFmpeg'}{/if}</button></div>
          <div class="update-card app-update"><div class="update-head"><div class="update-icon"><Sparkles size={24} /></div><span class="dependency-pill ready">APP</span></div><h2>getYT Downloader</h2><p>Novos recursos e melhorias de experiência chegam pelas releases do projeto.</p><div class="update-version"><span>Versão atual</span><strong>{state.version}</strong></div><button on:click={checkApp} disabled={checkingApp}>{#if checkingApp}<LoaderCircle size={17} class="spin" /> Verificando…{:else}<Search size={17} /> Verificar atualização{/if}</button></div>
        </div>
        {#if update}<div class="release-card"><div class="release-icon">{#if update.available}<Sparkles size={22} />{:else}<CheckCircle2 size={22} />{/if}</div><div><small>ÚLTIMA RELEASE</small><h3>{update.latest}{update.available ? ' está disponível' : ' · você está em dia'}</h3><p>{update.available ? 'A nova versão pode ser instalada agora.' : 'Seu aplicativo já está atualizado.'}</p></div>{#if update.available}{#if update.canInstall}<button class="release-button" on:click={installApp} disabled={installingApp || running}>{#if installingApp}<LoaderCircle size={16} class="spin" /> Preparando…{:else}Instalar atualização <ArrowRight size={16} />{/if}</button>{:else}<button class="release-button" on:click={openRelease}>Ver release <ArrowUpRight size={16} /></button>{/if}{/if}</div>{/if}
        <div class="update-note"><Info size={18} /><p>As atualizações do yt-dlp e FFmpeg são instaladas no seu perfil de usuário. A atualização do app usa os pacotes oficiais das releases deste repositório.</p></div>
      </div>
    {/if}
  </main>
</div>
