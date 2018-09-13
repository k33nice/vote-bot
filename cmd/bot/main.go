package main

import (
	"log"
	"time"

	vote "github.com/k33nice/vote-bot/pkg"
	"github.com/pkg/errors"
	tb "gopkg.in/tucnak/telebot.v2"
)

var cfg *vote.Config
var bot *vote.Bot

func init() {
	cfg = vote.NewConfig()
	b, err := vote.NewBot(tb.Settings{
		Token:  cfg.APIToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	}, cfg)

	if err != nil {
		log.Fatal(errors.Wrap(err, "cannot create new bot"))
		return
	}

	bot = b
}

func main() {
	bot.Handle("/help", func(m *tb.Message) {
		bot.Send(m.Chat, "```"+vote.HelpMessage+"```", tb.ModeMarkdown)
	})

	bot.Handle("/where", func(m *tb.Message) {
		loc := tb.Location{Lat: float32(cfg.Place.Location.Latitude), Lng: float32(cfg.Place.Location.Longitude)}
		loc.Send(&bot.Bot, m.Chat, &tb.SendOptions{})
	})

	bot.Handle("/info", func(m *tb.Message) {
		bot.Send(m.Chat, cfg.Place.URL)
	})

	bot.Handle("/result", func(m *tb.Message) {
		bot.Send(m.Chat, bot.GetVoteResult())
	})

	bot.Handle("/start", handleStart)

	bot.Handle(tb.OnAddedToGroup, handleStart)

	go func() {
		for {
			var un string
			var date time.Time

			switch m := bot.Pinned.(type) {
			case *tb.Message:
				date = m.Time()
				un = m.Sender.Username
			case *vote.PinnedMessage:
				un = m.From.Username
				date = time.Unix(int64(m.Date), 0)
			}

			curYear, curWeek := time.Now().ISOWeek()
			pinYear, pinWeek := date.ISOWeek()
			if curWeek != pinWeek && curYear != pinYear && un == bot.Me.Username {
				bot.UnpinMessage()
			}

			if bot.Pinned != nil && un == bot.Me.Username {
				bot.UpdateVote()
			} else {
				bot.CreateVote()
			}

			bot.CreateHandlers()
			time.Sleep(time.Second * 5)
		}
	}()

	bot.Start()
}

func handleStart(m *tb.Message) {
	if m.FromGroup() {
		bot.Channel = m.Chat

		pm, err := bot.GetPinnedMessage(int(m.Chat.ID))
		if err != nil {
			log.Panic(err)
		}
		if pm != nil {
			bot.Pinned = pm
		}
	}
}
