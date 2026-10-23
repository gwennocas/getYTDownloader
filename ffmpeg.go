package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const ffmpegZIP = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"

func (a *App) UpdateFFmpeg() (string, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return "", errors.New("Instalação automática do FFmpeg disponível para Windows x64")
	}
	if err := a.beginMaintenance(); err != nil {
		return "", err
	}
	defer a.endMaintenance()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ffmpegZIP+".sha256", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", fmt.Errorf("checksum do FFmpeg: HTTP %d", resp.StatusCode)
	}
	checksumBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(checksumBytes))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", errors.New("Checksum do FFmpeg inválido")
	}
	base, err := dataDir()
	if err != nil {
		return "", err
	}
	stage := filepath.Join(base, "ffmpeg-update")
	bin := filepath.Join(base, "bin")
	if err := os.MkdirAll(stage, 0700); err != nil {
		return "", err
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		return "", err
	}
	zipPath := filepath.Join(stage, "ffmpeg.zip")
	if err := downloadFile(ctx, ffmpegZIP, zipPath, 250<<20); err != nil {
		return "", err
	}
	if err := verifySHA256(zipPath, fields[0]); err != nil {
		return "", err
	}
	if err := extractFFmpeg(zipPath, bin); err != nil {
		return "", err
	}
	_ = os.Remove(zipPath)
	return "FFmpeg instalado e pronto para MP3 e vídeo em alta qualidade", nil
}

func extractFFmpeg(zipPath, bin string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	found := 0
	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name != "ffmpeg.exe" && name != "ffprobe.exe" {
			continue
		}
		if !strings.Contains(strings.ReplaceAll(f.Name, "\\", "/"), "/bin/") {
			continue
		}
		if f.UncompressedSize64 > 200<<20 {
			return errors.New("Binário do FFmpeg maior que o esperado")
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		dest := filepath.Join(bin, name)
		tmp, err := os.CreateTemp(bin, ".ffmpeg-*")
		if err != nil {
			src.Close()
			return err
		}
		_, copyErr := ioCopyLimited(tmp, src, 200<<20)
		src.Close()
		closeErr := tmp.Close()
		if copyErr != nil {
			os.Remove(tmp.Name())
			return copyErr
		}
		if closeErr != nil {
			os.Remove(tmp.Name())
			return closeErr
		}
		backup := dest + ".old"
		_ = os.Remove(backup)
		if _, err := os.Stat(dest); err == nil {
			if err := os.Rename(dest, backup); err != nil {
				os.Remove(tmp.Name())
				return err
			}
		}
		if err := os.Rename(tmp.Name(), dest); err != nil {
			_ = os.Rename(backup, dest)
			return err
		}
		_ = os.Remove(backup)
		found++
	}
	if found != 2 {
		return errors.New("Pacote do FFmpeg incompleto")
	}
	return nil
}
