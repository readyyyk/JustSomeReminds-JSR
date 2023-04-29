package main

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/readyyyk/terminal-todos-go/pkg/logs"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

var b *bot.Bot
var ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
var rutines = make(map[uuid.UUID]*chan bool)

func watcher(d chan bool, timeToSleep time.Duration, update *models.Update, text string) (exited bool) {
	select {
	case _ = <-d:
		return false
	case _ = <-time.After(timeToSleep):
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   text,
		})
		return true
	}
}

func stopHandler(update *models.Update) {
	for k, r := range rutines {
		*r <- true
		delete(rutines, k)
		fmt.Println(" -Stopped-> " + k.String())
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "All watchers are stopped",
	})
	return
}
func cancelHandler(update *models.Update, cid uuid.UUID, text string) {
	if rutines[cid] == nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Not found",
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Stopped watcher with uuid: " + text,
	})
	*rutines[cid] <- true
	delete(rutines, cid)
	fmt.Println(" -stopped-> " + text)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	l, _ := time.LoadLocation("Europe/Minsk")
	args := strings.Split(update.Message.Text, " ")

	if args[0] == "/stop" {
		stopHandler(update)
		return
	}
	if len(args) == 2 {
		if args[0] == "/cancel" {
			cid, err := uuid.Parse(args[1])
			if err != nil {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Wrong uuid",
				})
				return
			}
			cancelHandler(update, cid, args[1])
			return
		}
	}
	if len(args) >= 3 {
		sendDate, err := time.Parse("15:04", args[1])
		if err != nil {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Wrong time format (hh:mm)",
			})
			return
		}
		now := time.Now().UTC()
		respDate := time.Date(now.Year(), now.Month(), now.Day(), sendDate.Hour(), sendDate.Minute(), 0, 0, l)
		if respDate.Before(now) {
			respDate = respDate.Add(time.Hour * 24)
		}
		timeToSleep := time.Second * time.Duration(respDate.Unix()-now.Unix())

		if args[0] == "/add" {
			rutine := make(chan bool)
			cid := uuid.New()
			rutines[cid] = &rutine

			go func() {
				dontKill := true
				timeToSleepL := timeToSleep
				for dontKill {
					dontKill = watcher(rutine, timeToSleepL, update, strings.Join(args[2:], " "))
					timeToSleepL = time.Hour * 24
				}
			}()

			fmt.Println(" -new-> " + sendDate.Format("15:04") + "  -  " + args[2] + " on " + strconv.Itoa(int(update.Message.Chat.ID)))
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:              update.Message.Chat.ID,
				Text:                "watched ✅✅, to cancel type :",
				DisableNotification: true,
				ReplyToMessageID:    update.Message.ID,
			})
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:              update.Message.Chat.ID,
				Text:                "/cancel " + cid.String(),
				DisableNotification: true,
			})
			return
		}
		if args[0] == "/single" {
			fmt.Println(" -single-> " + sendDate.Format("15:04") + "  -  " + args[2])

			rutine := make(chan bool)
			cid := uuid.New()
			rutines[cid] = &rutine
			go watcher(rutine, timeToSleep, update, strings.Join(args[2:], " "))

			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:              update.Message.Chat.ID,
				Text:                "watched 👷🏿‍♂️👷🏿‍♂️, to cancel type :",
				DisableNotification: true,
				ReplyToMessageID:    update.Message.ID,
			})
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:              update.Message.Chat.ID,
				Text:                "/cancel " + cid.String(),
				DisableNotification: true,
			})
			return
		}
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Wrong input ❌",
	})
	return
}

func main() {
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	var err error
	logs.LogError(godotenv.Load(".env"))
	b, err = bot.New(os.Getenv("token"), opts...)
	if err != nil {
		panic(err)
	}
	logs.LogSuccess("Connected\n")

	b.Start(ctx)
}
