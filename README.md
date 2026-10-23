# getYTDownloader

App desktop para Windows feito com **Go + Wails v2 + Svelte**. Baixa vídeos, áudio e playlists com yt-dlp, com uma interface em azul escuro e dourado.

## O que faz

- Analisa links e identifica vídeos ou playlists antes de baixar.
- Baixa um vídeo ou uma playlist inteira. Playlists são organizadas em uma subpasta e usam um arquivo de histórico para retomar sem repetir itens concluídos.
- Oferece vídeo na melhor qualidade ou até 1080p, 720p ou 480p; áudio MP3; e áudio original sem conversão.
- Mostra progresso, velocidade, tempo estimado e atividade do yt-dlp. Permite cancelar e abrir a pasta de destino.
- Instala ou atualiza **yt-dlp** e **FFmpeg** pelo próprio app, no perfil do usuário.
- Verifica releases do app e instala uma nova versão publicada no GitHub. O pacote é conferido com o SHA-256 informado pela API de releases.

> Baixe apenas conteúdo que você tem permissão para salvar e respeite os termos do site de origem.

## Usar no Windows

1. Baixe `getYTDownloader_windows_amd64.zip` em [Releases](https://github.com/gwennocas/getYTDownloader/releases) e extraia `getYTDownloader.exe` para uma pasta onde você possa escrever.
2. Abra o aplicativo. Em **Atualizações**, instale o yt-dlp e o FFmpeg. O FFmpeg permite MP3 e vídeo com áudio separado em alta qualidade.
3. Na tela **Baixar mídia**, cole o link, escolha formato e pasta. Para playlists, deixe **Baixar playlist inteira** ativado.

O app requer o Microsoft WebView2 Runtime, incluído normalmente no Windows 10/11. Não é necessário instalar Go para usar uma release.

## Desenvolvimento

Pré-requisitos: [Go 1.26+](https://go.dev/dl/), [Node.js 24+](https://nodejs.org/) e [Wails v2](https://v2.wails.io/docs/gettingstarted/installation/). No Windows, confirme que o WebView2 Runtime está instalado.

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0
cd frontend
npm ci
cd ..
wails dev
```

Para gerar o executável:

```powershell
wails build -clean -platform windows/amd64 -o getYTDownloader.exe
```

Para validar:

```powershell
cd frontend
npm run check
npm run build
cd ..
go test ./...
```

O workflow de release gera `getYTDownloader_windows_amd64.zip` quando uma tag `v*` é enviada ao GitHub. A versão compilada é inserida no binário e é usada para verificar atualizações. Builds locais mostram `dev` e podem verificar releases, mas não fazem instalação automática por cima de um build de desenvolvimento.

## Onde ficam os arquivos

- Mídia: pasta escolhida na tela de download.
- Histórico de playlists: `.getyt-archive.txt` dentro da pasta de destino.
- Ferramentas e atualizações: pasta de configuração do usuário em `getYTDownloader`.

O projeto chama o yt-dlp como processo separado, sem executar URLs ou nomes de arquivo por um shell. O botão de atualização do app só usa pacotes da página de releases deste repositório.
