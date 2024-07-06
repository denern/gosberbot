package telegram

import (
	"fmt"
	"gosberbot/internal/domain"
	"log"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	FileBaseUrl = "https://api.telegram.org/file/bot"
)

type Client struct {
	bot   *tele.Bot
	token string
	queue chan domain.Message
}

func NewClient(token string, queue chan domain.Message) *Client {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &Client{
		bot:   bot,
		token: token,
		queue: queue,
	}
}

func (s *Client) SendMessage(msg domain.Message) {
	s.queue <- msg
}

func (s *Client) Send(user any, text string) error {
	if u, ok := user.(*tele.User); ok {
		_, err := s.bot.Send(u, text)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		return nil
	}

	return fmt.Errorf("failed to send message: invalid user type %T", user)
}

func (c *Client) Hello(ctx tele.Context) error {
	return ctx.Send("Hello!")
}

func (c *Client) OnText(ctx tele.Context) error {
	msg := domain.Message{
		Type:    "text",
		Payload: ctx.Text(),
		User:    ctx.Sender(),
	}

	c.SendMessage(msg)

	return nil
}

func (c *Client) OnVideo(ctx tele.Context) error {
	video := ctx.Message().Video

	file, err := c.bot.FileByID(video.FileID)

	if err != nil {
		return fmt.Errorf("onVideo error: %w", err)
	}

	msg := domain.Message{
		Type:    "video",
		Payload: FileBaseUrl + c.token + "/" + file.FilePath,
		User:    ctx.Sender(),
	}

	c.SendMessage(msg)

	return nil
}

func (c *Client) OnAudio(ctx tele.Context) error {
	audio := ctx.Message().Audio

	file, err := c.bot.FileByID(audio.FileID)

	if err != nil {
		return fmt.Errorf("onAudio error: %w", err)
	}

	msg := domain.Message{
		Type:    "audio",
		Payload: FileBaseUrl + c.token + "/" + file.FilePath,
		User:    ctx.Sender(),
	}

	c.SendMessage(msg)

	return nil
}

func (c *Client) OnVoice(ctx tele.Context) error {
	voice := ctx.Message().Voice

	file, err := c.bot.FileByID(voice.FileID)

	if err != nil {
		return fmt.Errorf("onVoice error: %w", err)
	}

	msg := domain.Message{
		Type:    "voice",
		Payload: FileBaseUrl + c.token + "/" + file.FilePath,
		User:    ctx.Sender(),
	}

	c.SendMessage(msg)

	return nil
}

func (c *Client) Start() {
	c.bot.Handle("/hello", func(ctx tele.Context) error {
		return c.Hello(ctx)
	})

	c.bot.Handle(tele.OnText, func(ctx tele.Context) error {
		fmt.Printf("OnText\n")
		return c.OnText(ctx)
	})

	c.bot.Handle(tele.OnVideo, func(ctx tele.Context) error {
		fmt.Printf("OnVideo\n")
		return c.OnVideo(ctx)
	})

	c.bot.Handle(tele.OnAudio, func(ctx tele.Context) error {
		fmt.Printf("OnAudio\n")
		return c.OnAudio(ctx)
	})

	c.bot.Handle(tele.OnVoice, func(ctx tele.Context) error {
		fmt.Printf("OnVoice\n")
		return c.OnVoice(ctx)
	})

	c.bot.Start()
}

func (c *Client) Stop() {
	c.bot.Stop()
}
