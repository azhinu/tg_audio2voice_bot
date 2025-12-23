package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	supportedMIMEs = map[string]struct{}{
		"audio/mpeg":       {},
		"audio/mp3":        {},
		"audio/mp4":        {},
		"audio/aac":        {},
		"audio/x-m4a":      {},
		"audio/ogg":        {},
		"audio/vorbis":     {},
		"application/ogg":  {},
		"application/opus": {},
	}
	supportedExts = []string{".mp3", ".m4a", ".aac", ".ogg", ".oga", ".opus", ".wav", ".flac", ".webm"}
)

func extractAudio(msg *tgbotapi.Message) (fileID, fileName string, err error) {
	if msg.Audio != nil {
		return msg.Audio.FileID, msg.Audio.FileName, nil
	}

	if doc := msg.Document; doc != nil {
		if isSupported(doc.FileName, doc.MimeType) {
			return doc.FileID, doc.FileName, nil
		}
		return "", "", errors.New("unsupported document type")
	}

	return "", "", errors.New("no audio in message")
}

func isSupported(name, mime string) bool {
	for _, ext := range supportedExts {
		if strings.EqualFold(filepath.Ext(name), ext) {
			return true
		}
	}
	if mime == "" {
		return false
	}
	if _, ok := supportedMIMEs[strings.ToLower(mime)]; ok {
		return true
	}
	return false
}

func downloadTelegramFile(ctx context.Context, bot *tgbotapi.BotAPI, fileID, fileName string) (string, error) {
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("get file: %w", err)
	}

	link := file.Link(bot.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %s", resp.Status)
	}

	ext := filepath.Ext(file.FilePath)
	if ext == "" {
		ext = filepath.Ext(fileName)
	}
	if ext == "" {
		ext = ".audio"
	}

	tmpFile, err := os.CreateTemp("", "tg-audio-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err = io.Copy(tmpFile, resp.Body); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

func convertToVoice(ctx context.Context, inputPath string) (string, error) {
	outputFile, err := os.CreateTemp("", "tg-voice-*.ogg")
	if err != nil {
		return "", fmt.Errorf("create output: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-vn",
		"-ac", "1",
		"-ar", "48000",
		"-c:a", "libopus",
		outputPath,
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("ffmpeg: %v: %s", err, errMsg)
		}
		return "", fmt.Errorf("ffmpeg: %w", err)
	}

	return outputPath, nil
}

func getAudioDuration(ctx context.Context, path string) (int, error) {
	probeCmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	probeOut, err := probeCmd.Output()
	if err != nil {
		return 0, err
	}
	durStr := strings.TrimSpace(string(probeOut))
	if durStr == "" {
		return 0, fmt.Errorf("duration not found")
	}
	f, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}
