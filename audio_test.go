package main

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestIsSupportedByExtension(t *testing.T) {
	if !isSupported("track.MP3", "") {
		t.Fatalf("expected .mp3 to be supported")
	}
}

func TestIsSupportedByMIME(t *testing.T) {
	if !isSupported("unknown.bin", "audio/mpeg") {
		t.Fatalf("expected audio/mpeg mime to be supported")
	}
}

func TestIsSupportedUnknown(t *testing.T) {
	if isSupported("file.txt", "text/plain") {
		t.Fatalf("text/plain should not be supported")
	}
}

func TestExtractAudioFromAudio(t *testing.T) {
	msg := &tgbotapi.Message{
		Audio: &tgbotapi.Audio{
			FileID:   "123",
			FileName: "song.mp3",
		},
	}
	fileID, fileName, err := extractAudio(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileID != "123" || fileName != "song.mp3" {
		t.Fatalf("unexpected result: %s %s", fileID, fileName)
	}
}

func TestExtractAudioFromDocument(t *testing.T) {
	msg := &tgbotapi.Message{
		Document: &tgbotapi.Document{
			FileID:   "doc",
			FileName: "voice.ogg",
			MimeType: "audio/ogg",
		},
	}
	fileID, fileName, err := extractAudio(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fileID != "doc" || fileName != "voice.ogg" {
		t.Fatalf("unexpected result: %s %s", fileID, fileName)
	}
}

func TestExtractAudioUnsupportedDocument(t *testing.T) {
	msg := &tgbotapi.Message{
		Document: &tgbotapi.Document{
			FileID:   "doc",
			FileName: "notes.txt",
			MimeType: "text/plain",
		},
	}
	if _, _, err := extractAudio(msg); err == nil {
		t.Fatalf("expected error for unsupported document")
	}
}

func TestExtractAudioNoMedia(t *testing.T) {
	msg := &tgbotapi.Message{}
	if _, _, err := extractAudio(msg); err == nil {
		t.Fatalf("expected error when no media present")
	}
}
