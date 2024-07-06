package main

import (
	"context"
	"fmt"
	"gosberbot/internal/domain"
	"gosberbot/internal/provider/gigachat"
	"gosberbot/internal/provider/salutespeech"
	"gosberbot/internal/provider/telegram"
	"gosberbot/internal/service"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	queue := make(chan domain.Message, 10)

	bot := telegram.NewClient(os.Getenv("BOT_TOKEN"), queue)
	if bot == nil {
		fmt.Printf("telegram bot error\n")
		return
	}

	speech := salutespeech.NewClient(os.Getenv("SALUTESPEECH_AUTH_KEY"))
	chat := gigachat.NewClient(os.Getenv("GIGACHAT_AUTH_KEY"))

	if err := speech.GetToken(); err != nil {
		fmt.Printf("salutespeech error: %v\n", err)
		return
	}

	if err := chat.GetToken(); err != nil {
		fmt.Printf("gigachat error: %v\n", err)
		return
	}

	go func() {
		bot.Start()
	}()

	fmt.Printf("Start service\n")

	srv := service.NewService(queue)

	srv.Init(bot, speech, chat)

	go func() {
		srv.Start(ctx)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(
		quit,
		syscall.SIGABRT, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGTERM, os.Interrupt,
	)

	fmt.Printf("Caught signal %s. Shutting down...\n", <-quit)

	cancel()

	srv.Stop()
	bot.Stop()

	close(queue)
}
