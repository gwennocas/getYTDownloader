package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var version = "dev"

type App struct {
	ctx      context.Context
	mu       sync.Mutex
	cancel   context.CancelFunc
	active   bool
	updating bool
}

type State struct {
	Version      string `json:"version"`
	YTDLPVersion string `json:"ytdlpVersion"`
	YTDLPReady   bool   `json:"ytdlpReady"`
	FFmpegReady  bool   `json:"ffmpegReady"`
	DownloadPath string `json:"downloadPath"`
	Active       bool   `json:"active"`
}

type Preview struct {
	Title       string `json:"title"`
	Uploader    string `json:"uploader"`
	Thumbnail   string `json:"thumbnail"`
	Duration    int    `json:"duration"`
	IsPlaylist  bool   `json:"isPlaylist"`
	EntryCount  int    `json:"entryCount"`
	Description string `json:"description"`
}

type DownloadOptions struct {
	URL      string `json:"url"`
	Folder   string `json:"folder"`
	Mode     string `json:"mode"`
	Quality  string `json:"quality"`
	Playlist bool   `json:"playlist"`
}

type DownloadEvent struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Percent string `json:"percent"`
	Speed   string `json:"speed"`
	ETA     string `json:"eta"`
	Index   string `json:"index"`
	Count   string `json:"count"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.mu.Unlock()
}

func (a *App) emit(event DownloadEvent) {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "download", event)
	}
}

func dataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "getYTDownloader"), nil
}

func ytDLPPath() (string, error) {
	base, err := dataDir()
	if err != nil {
		return "", err
	}
	name := "yt-dlp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	local := filepath.Join(base, "bin", name)
	if _, err := os.Stat(local); err == nil {
		return local, nil
	}
	return exec.LookPath("yt-dlp")
}

func ffmpegPath() string {
	if base, err := dataDir(); err == nil {
		name := "ffmpeg"
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		local := filepath.Join(base, "bin", name)
		if _, err := os.Stat(local); err == nil {
			return local
		}
	}
	path, _ := exec.LookPath("ffmpeg")
	return path
}

func defaultDownloadPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, "Downloads")
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	return home
}

func (a *App) GetState() State {
	path, err := ytDLPPath()
	state := State{Version: version, FFmpegReady: ffmpegPath() != "", DownloadPath: defaultDownloadPath()}
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, path, "--version").Output()
		if err == nil {
			state.YTDLPVersion = strings.TrimSpace(string(out))
			state.YTDLPReady = true
		}
	}
	a.mu.Lock()
	state.Active = a.active
	a.mu.Unlock()
	return state
}

func (a *App) ChooseFolder() (string, error) {
	if a.ctx == nil {
		return "", errors.New("janela ainda não está pronta")
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: "Escolher pasta de downloads", DefaultDirectory: defaultDownloadPath()})
}

func validateURL(raw string) error {
	if len(raw) > 2048 {
		return errors.New("URL muito longa")
	}
	u, err := parseWebURL(raw)
	if err != nil || u.Hostname() == "" {
		return errors.New("Cole uma URL http ou https válida")
	}
	return nil
}

func (a *App) Inspect(raw string) (Preview, error) {
	if err := validateURL(raw); err != nil {
		return Preview{}, err
	}
	path, err := ytDLPPath()
	if err != nil {
		return Preview{}, errors.New("Instale o yt-dlp para analisar o link")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--ignore-config", "--no-warnings", "--flat-playlist", "--dump-single-json", "--skip-download", raw)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return Preview{}, errors.New("A análise demorou demais. Tente novamente")
		}
		return Preview{}, errors.New("Não foi possível analisar o link. Confira a URL e sua conexão")
	}
	var info struct {
		Title       string            `json:"title"`
		Uploader    string            `json:"uploader"`
		Thumbnail   string            `json:"thumbnail"`
		Duration    int               `json:"duration"`
		Description string            `json:"description"`
		Type        string            `json:"_type"`
		Entries     []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return Preview{}, errors.New("Resposta inesperada do yt-dlp")
	}
	thumbnail := info.Thumbnail
	if !strings.HasPrefix(thumbnail, "https://") {
		thumbnail = ""
	}
	return Preview{Title: info.Title, Uploader: info.Uploader, Thumbnail: thumbnail, Duration: info.Duration, IsPlaylist: info.Type == "playlist" || len(info.Entries) > 0, EntryCount: len(info.Entries), Description: info.Description}, nil
}

func (a *App) StartDownload(opts DownloadOptions) error {
	if err := validateURL(opts.URL); err != nil {
		return err
	}
	if opts.Mode != "video" && opts.Mode != "mp3" && opts.Mode != "original" {
		return errors.New("Formato inválido")
	}
	if opts.Quality != "best" && opts.Quality != "1080" && opts.Quality != "720" && opts.Quality != "480" {
		return errors.New("Qualidade inválida")
	}
	path, err := ytDLPPath()
	if err != nil {
		return errors.New("Instale o yt-dlp antes de baixar")
	}
	folder, err := filepath.Abs(opts.Folder)
	if err != nil {
		return errors.New("Pasta inválida")
	}
	info, err := os.Stat(folder)
	if err != nil || !info.IsDir() {
		return errors.New("Escolha uma pasta existente")
	}
	if opts.Mode == "mp3" && ffmpegPath() == "" {
		return errors.New("MP3 precisa do FFmpeg instalado. Use áudio original ou instale o FFmpeg")
	}

	a.mu.Lock()
	if a.active {
		a.mu.Unlock()
		return errors.New("Já existe um download em andamento")
	}
	if a.updating {
		a.mu.Unlock()
		return errors.New("Espere a atualização terminar antes de baixar")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.active, a.cancel = true, cancel
	a.mu.Unlock()
	go a.runDownload(ctx, path, folder, opts)
	return nil
}

func (a *App) CancelDownload() {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.mu.Unlock()
}

func (a *App) runDownload(ctx context.Context, path, folder string, opts DownloadOptions) {
	defer func() {
		a.mu.Lock()
		a.active, a.cancel = false, nil
		a.mu.Unlock()
	}()
	args := buildDownloadArgs(folder, opts, ffmpegPath())
	cmd := exec.CommandContext(ctx, path, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		a.emit(DownloadEvent{Status: "error", Message: err.Error()})
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		a.emit(DownloadEvent{Status: "error", Message: err.Error()})
		return
	}
	if err := cmd.Start(); err != nil {
		a.emit(DownloadEvent{Status: "error", Message: err.Error()})
		return
	}
	a.emit(DownloadEvent{Status: "running", Message: "Preparando download…"})
	var wg sync.WaitGroup
	wg.Add(2)
	read := func(scanner *bufio.Scanner) {
		defer wg.Done()
		scanner.Buffer(make([]byte, 4096), 2*1024*1024)
		for scanner.Scan() {
			a.handleLine(scanner.Text())
		}
	}
	go read(bufio.NewScanner(stdout))
	go read(bufio.NewScanner(stderr))
	wg.Wait()
	err = cmd.Wait()
	if ctx.Err() != nil {
		a.emit(DownloadEvent{Status: "cancelled", Message: "Download cancelado"})
		return
	}
	if err != nil {
		a.emit(DownloadEvent{Status: "error", Message: "O download falhou. Confira os detalhes acima"})
		return
	}
	a.emit(DownloadEvent{Status: "done", Message: "Download concluído"})
}

func buildDownloadArgs(folder string, opts DownloadOptions, ffmpeg string) []string {
	args := []string{"--ignore-config", "--newline", "--no-overwrites", "--windows-filenames", "--progress-template", "download:GETYT_PROGRESS:%(progress._percent_str)s|%(progress._speed_str)s|%(progress._eta_str)s|%(info.playlist_index)s|%(info.playlist_count)s", "--print", "before_dl:GETYT_ITEM:%(title)s"}
	if opts.Playlist {
		args = append(args, "--yes-playlist", "--download-archive", filepath.Join(folder, ".getyt-archive.txt"))
		args = append(args, "-o", filepath.Join(folder, "%(playlist_title|Playlist)s", "%(playlist_index|0)03d - %(title).180B [%(id)s].%(ext)s"))
	} else {
		args = append(args, "--no-playlist", "-o", filepath.Join(folder, "%(title).180B [%(id)s].%(ext)s"))
	}
	if ffmpeg != "" {
		args = append(args, "--ffmpeg-location", filepath.Dir(ffmpeg))
	}
	switch opts.Mode {
	case "mp3":
		args = append(args, "-x", "--audio-format", "mp3", "--audio-quality", "0")
	case "original":
		args = append(args, "-f", "bestaudio/best")
	case "video":
		format := "bv*+ba/b"
		if ffmpeg == "" {
			format = "best[ext=mp4]/best"
		}
		if opts.Quality != "best" {
			format = fmt.Sprintf("bv*[height<=%s]+ba/b[height<=%s]", opts.Quality, opts.Quality)
			if ffmpeg == "" {
				format = fmt.Sprintf("best[height<=%s][ext=mp4]/best[height<=%s]", opts.Quality, opts.Quality)
			}
		}
		args = append(args, "-f", format)
		if ffmpeg != "" {
			args = append(args, "--merge-output-format", "mp4")
		}
	}
	args = append(args, "--", opts.URL)
	return args
}

func (a *App) handleLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if strings.HasPrefix(line, "GETYT_PROGRESS:") {
		parts := strings.SplitN(strings.TrimPrefix(line, "GETYT_PROGRESS:"), "|", 5)
		for len(parts) < 5 {
			parts = append(parts, "")
		}
		a.emit(DownloadEvent{Status: "progress", Percent: strings.TrimSpace(parts[0]), Speed: strings.TrimSpace(parts[1]), ETA: strings.TrimSpace(parts[2]), Index: strings.TrimSpace(parts[3]), Count: strings.TrimSpace(parts[4])})
		return
	}
	if strings.HasPrefix(line, "GETYT_ITEM:") {
		a.emit(DownloadEvent{Status: "item", Message: strings.TrimPrefix(line, "GETYT_ITEM:")})
		return
	}
	if len(line) > 500 {
		line = line[:500] + "…"
	}
	a.emit(DownloadEvent{Status: "log", Message: line})
}

func (a *App) OpenFolder(folder string) error {
	info, err := os.Stat(folder)
	if err != nil || !info.IsDir() {
		return errors.New("Pasta não encontrada")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", folder).Start()
	case "darwin":
		return exec.Command("open", folder).Start()
	default:
		return exec.Command("xdg-open", folder).Start()
	}
}

func (a *App) UpdateYTDLP() (string, error) {
	if err := a.beginMaintenance(); err != nil {
		return "", err
	}
	defer a.endMaintenance()
	if runtime.GOOS != "windows" {
		return "", errors.New("Instalação automática do yt-dlp está disponível no Windows")
	}
	base, err := dataDir()
	if err != nil {
		return "", err
	}
	bin := filepath.Join(base, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		return "", err
	}
	dest := filepath.Join(bin, "yt-dlp.exe")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := downloadFile(ctx, "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe", dest, 150<<20); err != nil {
		return "", err
	}
	out, err := exec.Command(dest, "--version").Output()
	if err != nil {
		return "", errors.New("yt-dlp baixado, mas não conseguiu iniciar")
	}
	return strings.TrimSpace(string(out)), nil
}

func (a *App) beginMaintenance() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.active {
		return errors.New("Espere o download terminar antes de atualizar")
	}
	if a.updating {
		return errors.New("Outra atualização já está em andamento")
	}
	a.updating = true
	return nil
}

func (a *App) endMaintenance() {
	a.mu.Lock()
	a.updating = false
	a.mu.Unlock()
}

func downloadFile(ctx context.Context, url, dest string, maxSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "getYTDownloader/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("falha na conexão: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("servidor retornou HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxSize {
		return errors.New("arquivo maior que o esperado")
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".getyt-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	n, err := ioCopyLimited(tmp, resp.Body, maxSize)
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n == 0 {
		return errors.New("download vazio")
	}
	backup := dest + ".old"
	_ = os.Remove(backup)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Rename(backup, dest)
		return err
	}
	_ = os.Remove(backup)
	return nil
}
