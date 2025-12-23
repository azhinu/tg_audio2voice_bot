package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleMessage(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg.IsCommand() {
		handleCommand(bot, msg)
		return
	}

	fileID, fileName, err := extractAudio(msg)
	if err != nil {
		replyText(bot, msg.Chat.ID, msg.MessageID, "Send an audio file (mp3, m4a, ogg/vorbis) and I'll convert it to a voice message.")
		return
	}

	replyText(bot, msg.Chat.ID, msg.MessageID, "Processing…")
	submitConversionJob(conversionJob{
		ctx:      ctx,
		bot:      bot,
		msg:      msg,
		fileID:   fileID,
		fileName: fileName,
	})
}

func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	info := fmt.Sprintf("Send an audio file (mp3, m4a, ogg/vorbis) and I'll convert it to a voice message. Sources are available at <a href=\"%s\">GitHub</a>.", repoURL)
	switch msg.Command() {
	case "start", "help":
		replyText(bot, msg.Chat.ID, msg.MessageID, info, "HTML")
	default:
		replyText(bot, msg.Chat.ID, msg.MessageID, "Unknown command. Just send an audio file.")
	}
}

func replyText(bot *tgbotapi.BotAPI, chatID int64, replyTo int, text string, parseMode ...string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyTo
	if len(parseMode) > 0 {
		msg.ParseMode = parseMode[0]
	}
	if _, err := bot.Send(msg); err != nil {
		log.Printf("send message error: %v", err)
	}
}

type conversionJob struct {
	ctx      context.Context
	bot      *tgbotapi.BotAPI
	msg      *tgbotapi.Message
	fileID   string
	fileName string
}

const maxConversionQueueSize = 100

var (
	conversionQueue chan conversionJob
	queueMu         sync.Mutex
)

func submitConversionJob(job conversionJob) {
	queue := ensureConversionWorkers(job.ctx)
	select {
	case queue <- job:
	default:
		replyText(job.bot, job.msg.Chat.ID, job.msg.MessageID, "Bot is overloaded. Please try again in a couple of minutes.")
	}
}

func ensureConversionWorkers(ctx context.Context) chan conversionJob {
	queueMu.Lock()
	defer queueMu.Unlock()

	if conversionQueue != nil {
		return conversionQueue
	}

	workerTotal := runtime.GOMAXPROCS(0)
	if workerTotal < 1 {
		workerTotal = 1
	}

	queue := make(chan conversionJob, maxConversionQueueSize)
	for i := 0; i < workerTotal; i++ {
		go conversionWorker(queue)
	}

	go func(done <-chan struct{}, q chan conversionJob) {
		<-done
		queueMu.Lock()
		if conversionQueue == q {
			close(q)
			conversionQueue = nil
		} else {
			close(q)
		}
		queueMu.Unlock()
	}(ctx.Done(), queue)

	conversionQueue = queue
	return conversionQueue
}

func conversionWorker(queue <-chan conversionJob) {
	for job := range queue {
		handleConversionJob(job)
	}
}

func handleConversionJob(job conversionJob) {
	jobCtx, cancel := context.WithTimeout(job.ctx, defaultTimeout)
	defer cancel()

	inputPath, err := downloadTelegramFile(jobCtx, job.bot, job.fileID, job.fileName)
	if err != nil {
		log.Printf("download error: %v", err)
		replyText(job.bot, job.msg.Chat.ID, job.msg.MessageID, "Failed to download the file. Please try again. Files up to 20MB are supported.")
		return
	}
	defer os.Remove(inputPath)

	voicePath, err := convertToVoice(jobCtx, inputPath)
	if err != nil {
		log.Printf("convert error: %v", err)
		replyText(job.bot, job.msg.Chat.ID, job.msg.MessageID, fmt.Sprintf("Failed to convert the file to a voice message: %v", err))
		return
	}
	defer os.Remove(voicePath)

	voiceDuration, err := getAudioDuration(jobCtx, voicePath)
	if err != nil {
		log.Printf("get duration error: %v", err)
		replyText(job.bot, job.msg.Chat.ID, job.msg.MessageID, "Could not determine the duration of the voice message.")
		voiceDuration = 0
	}

	voice := tgbotapi.NewVoice(job.msg.Chat.ID, tgbotapi.FilePath(voicePath))
	voice.ReplyToMessageID = job.msg.MessageID
	voice.Duration = voiceDuration
	if _, err = job.bot.Send(voice); err != nil {
		log.Printf("send voice error: %v", err)
		replyText(job.bot, job.msg.Chat.ID, job.msg.MessageID, "Could not send the voice message.")
	}
}
