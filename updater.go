package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const releaseAPI = "https://api.github.com/repos/gwennocas/getYTDownloader/releases/latest"

type releaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}

type releaseData struct {
	Tag    string         `json:"tag_name"`
	URL    string         `json:"html_url"`
	Body   string         `json:"body"`
	Assets []releaseAsset `json:"assets"`
}

type UpdateInfo struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Available  bool   `json:"available"`
	CanInstall bool   `json:"canInstall"`
	URL        string `json:"url"`
	Notes      string `json:"notes"`
}

func fetchLatestRelease(ctx context.Context) (releaseData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseAPI, nil)
	if err != nil {
		return releaseData{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "getYTDownloader/"+version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return releaseData{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return releaseData{}, errors.New("Ainda não há uma versão publicada no GitHub")
	}
	if resp.StatusCode != http.StatusOK {
		return releaseData{}, fmt.Errorf("GitHub retornou HTTP %d", resp.StatusCode)
	}
	var release releaseData
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return releaseData{}, err
	}
	if release.Tag == "" {
		return releaseData{}, errors.New("Release sem número de versão")
	}
	return release, nil
}

func targetAsset(release releaseData) (releaseAsset, bool) {
	name := fmt.Sprintf("getYTDownloader_%s_%s.zip", runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if asset.Name == name && strings.HasPrefix(asset.URL, "https://github.com/gwennocas/getYTDownloader/releases/download/") {
			return asset, true
		}
	}
	return releaseAsset{}, false
}

func versionParts(s string) []int {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	parts := strings.SplitN(s, "-", 2)
	segments := strings.Split(parts[0], ".")
	result := make([]int, 3)
	for i := 0; i < len(segments) && i < 3; i++ {
		result[i], _ = strconv.Atoi(segments[i])
	}
	return result
}

func newerVersion(latest, current string) bool {
	if current == "dev" {
		return true
	}
	a, b := versionParts(latest), versionParts(current)
	for i := range a {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
	}
	return false
}

func (a *App) CheckAppUpdate() (UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return UpdateInfo{}, err
	}
	_, available := targetAsset(release)
	notes := release.Body
	if len(notes) > 1200 {
		notes = notes[:1200] + "…"
	}
	return UpdateInfo{Current: version, Latest: release.Tag, Available: newerVersion(release.Tag, version), CanInstall: runtime.GOOS == "windows" && available && version != "dev", URL: release.URL, Notes: notes}, nil
}

func (a *App) InstallAppUpdate() (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("A atualização automática está disponível no Windows")
	}
	if version == "dev" {
		return "", errors.New("A instalação automática só funciona na versão publicada")
	}
	if err := a.beginMaintenance(); err != nil {
		return "", err
	}
	defer a.endMaintenance()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return "", err
	}
	if !newerVersion(release.Tag, version) {
		return "", errors.New("Você já tem a versão mais recente")
	}
	asset, ok := targetAsset(release)
	if !ok {
		return "", errors.New("Esta release não tem pacote para este sistema")
	}
	if !strings.HasPrefix(asset.Digest, "sha256:") {
		return "", errors.New("Pacote sem checksum SHA-256")
	}
	base, err := dataDir()
	if err != nil {
		return "", err
	}
	stage := filepath.Join(base, "updates", release.Tag)
	if err := os.MkdirAll(stage, 0700); err != nil {
		return "", err
	}
	zipPath := filepath.Join(stage, asset.Name)
	if err := downloadFile(ctx, asset.URL, zipPath, 200<<20); err != nil {
		return "", err
	}
	if err := verifySHA256(zipPath, strings.TrimPrefix(asset.Digest, "sha256:")); err != nil {
		return "", err
	}
	newExe := filepath.Join(stage, "getYTDownloader.exe")
	if err := extractAppExe(zipPath, newExe); err != nil {
		return "", err
	}
	currentExe, err := os.Executable()
	if err != nil {
		return "", err
	}
	scriptPath := filepath.Join(stage, "install.ps1")
	script := fmt.Sprintf("$ErrorActionPreference = 'Stop'\n$source = '%s'\n$target = '%s'\n$pidToWait = %d\ntry { Wait-Process -Id $pidToWait -Timeout 30 -ErrorAction SilentlyContinue } catch {}\nStart-Sleep -Milliseconds 700\nCopy-Item -LiteralPath $source -Destination $target -Force\nStart-Process -FilePath $target\n", psQuote(newExe), psQuote(currentExe), os.Getpid())
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		return "", err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.SysProcAttr = updaterSysProcAttr()
	if err := cmd.Start(); err != nil {
		return "", err
	}
	if a.ctx != nil {
		wailsruntime.Quit(a.ctx)
	}
	return "Atualização preparada. O app vai reiniciar.", nil
}

func psQuote(s string) string { return strings.ReplaceAll(s, "'", "''") }

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), expected) {
		return errors.New("Checksum do pacote não confere")
	}
	return nil
}

func extractAppExe(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != "getYTDownloader.exe" {
			continue
		}
		if f.UncompressedSize64 > 200<<20 {
			return errors.New("Executável maior que o esperado")
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		defer src.Close()
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		_, copyErr := ioCopyLimited(out, src, 200<<20)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	return errors.New("Pacote sem getYTDownloader.exe")
}
