package service

import (
	"context"
	"fmt"
	"gosberbot/internal/domain"
	"gosberbot/internal/provider/gigachat"
	"gosberbot/internal/provider/salutespeech"
	"gosberbot/internal/provider/telegram"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type Service struct {
	speech *salutespeech.Client
	chat   *gigachat.Client
	bot    *telegram.Client
	queue  chan domain.Message
}

func NewService(queue chan domain.Message) *Service {
	return &Service{queue: queue}
}

func (s *Service) Init(bot *telegram.Client, speech *salutespeech.Client, chat *gigachat.Client) {
	s.bot = bot
	s.speech = speech
	s.chat = chat
}

func (s *Service) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.queue:
			s.processor(msg)
		}
	}
}

func (s *Service) Stop() {
}

func (s *Service) processor(msg domain.Message) {
	switch msg.Type {
	case "text":
		s.onText(msg)
	case "video":
		s.onVideo(msg)
	case "voice":
		s.onVoice(msg)
	case "audio":
		s.onAudio(msg)
	case "command":
		s.onCommand(msg)
	default:
		fmt.Printf("unknown message: %v\n", msg)
	}
}

func (s *Service) onText(msg domain.Message) {
	fmt.Printf("onText: %v\n", msg)

	text, err := s.chat.GetCompletions(msg.Payload)

	if err != nil {
		fmt.Printf("GetCompletions error: %v\n", err)
		return
	}

	fmt.Printf("Completion: %s\n", text)

	s.bot.Send(msg.User, text)
}

func (s *Service) onVideo(msg domain.Message) {
	fmt.Printf("onVideo: %v\n", msg)

	s.bot.Send(msg.User, "not implemented yet")
}

func (s *Service) onAudio(msg domain.Message) {
	fmt.Printf("onAudio: %v\n", msg)

	s.bot.Send(msg.User, "not implemented yet")
}

func (s *Service) onVoice(msg domain.Message) {
	fmt.Printf("onVoice: %v\n", msg)

	fileUrl := msg.Payload
	fileName := getFilename(fileUrl)

	if err := downloadFile(fileUrl, fileName); err != nil {
		fmt.Printf("downloadFile error\n")
		return
	}

	s.bot.Send(msg.User, fmt.Sprintf("Download %s", fileName))

	fileReqId, err := s.speech.UploadFile(fileName)

	if err != nil {
		fmt.Printf("UploadFile error: %v\n", err)
		return
	}

	s.bot.Send(msg.User, "Start recognize...")

	taskId, err := s.speech.RecognizeFile(fileReqId)

	if err != nil {
		fmt.Printf("RecognizeFile error: %v\n", err)
		return
	}

	for i := 0; i < 10; i++ {
		fileReqId, err = s.speech.GetStatus(taskId)

		if err != nil {
			fmt.Printf("GetStatus error: %v\n", err)
			return
		}

		if fileReqId != "" {
			break
		}

		time.Sleep(3 * time.Second)
	}

	s.bot.Send(msg.User, "Get text...")

	text, err := s.speech.DownloadFile(fileReqId)

	if err != nil {
		fmt.Printf("DownloadFile error: %v\n", err)
		return
	}

	s.bot.Send(msg.User, fmt.Sprintf("Text: %s\n", text))
}

func (s *Service) onCommand(msg domain.Message) {
	fmt.Printf("onCommand: %v\n", msg)
}

func (s *Service) Send(msg domain.Message) {
	s.queue <- msg
}

func getFilename(urlstr string) string {
	u, err := url.Parse(urlstr)
	if err != nil {
		fmt.Printf("Error due to parsing url: %v\n", err)
		return ""
	}

	x, _ := url.QueryUnescape(u.EscapedPath())

	return filepath.Base(x)
}

func downloadFile(urlstr, filename string) error {
	resp, err := http.Get(urlstr)

	if err != nil {
		return fmt.Errorf("http.Get error: %w", err)
	}

	defer resp.Body.Close()

	w, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)

	if err != nil {
		return fmt.Errorf("os.OpenFile error: %w", err)
	}

	defer w.Close()

	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("io.Copy error: %w", err)
	}

	return nil
}
