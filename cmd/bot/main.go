package main

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
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

	bot.Handle("/start_on_channel", handleStartOnChannel)

	bot.Handle("/set_date", handleSetDate)
	bot.Handle("/create", handleCreate)

	bot.Handle(tb.OnAddedToGroup, handleStart)

	go func() {
		for {
			log.Println("***BOT LOOP***")
			var un string
			var date time.Time

			now := time.Now()
			switch m := bot.Pinned.(type) {
			case *tb.Message:
				date = m.Time()
				un = m.Sender.Username
			case *vote.PinnedMessage:
				un = m.From.Username
				date = time.Unix(int64(m.Date), 0)
			}

			curYear, curWeek := now.ISOWeek()
			pinYear, pinWeek := date.ISOWeek()
			if (curWeek != pinWeek || curYear != pinYear) && now.Hour() == 16 && un == bot.Me.Username {
				log.Println("Unpin message")

				err := bot.UnpinMessage()
				if err != nil {
					log.Printf("cannot upin, err: %v", err)
				}
			}

			if bot.Pinned != nil && un == bot.Me.Username && (curWeek == pinWeek && curYear == pinYear) {
				log.Println("Update vote")
				if err := bot.UpdateVote(); err != nil {
					log.Printf("caught err: %s", err)
				}
			}

			if bot.Pinned == nil {
				log.Println("Create vote")
				if err := bot.CreateVote(); err != nil {
					log.Printf("caught error: %s", err)
				}
			}

			if int(now.Weekday()) == bot.Config.Weekday-1 && now.Hour() == 20 && now.Minute() == 0 {
				log.Println("send reminder")
				// bot.SendReminder()
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

func handleStartOnChannel(m *tb.Message) {
	if strings.ToLower(m.Sender.Username) != bot.Config.God {
		log.Printf("Cannot start need: %s, got: %s", m.Sender.Username, bot.Config.God)
		return
	}

	log.Println("HANDLE START")
	chatID := m.Payload

	chat, _ := bot.ChatByID(chatID)

	bot.Channel = chat

	pm, err := bot.GetPinnedMessage(int(chat.ID))
	if err != nil {
		log.Printf("handle start err: %s", err)
	}
	if pm == nil {
		log.Printf("no pinned message found")
		return
	}

	log.Printf("pm = %+v\n", pm)

	if pm != nil {
		bot.Pinned = pm
	}

	bot.Send(m.Sender, fmt.Sprintf("chatID: %d", chat.ID))
}

func handleSetDate(m *tb.Message) {
	allowedUsers := make(map[string]bool, len(bot.Config.Admins))
	for _, admin := range bot.Config.Admins {
		allowedUsers[admin] = true
	}

	sender := strings.ToLower(m.Sender.Username)
	if _, ok := allowedUsers[sender]; !ok {
		bot.Send(m.Sender, "Permission denied, Пёс")
		return
	}

	allowedDays := []string{
		time.Sunday.String(), time.Monday.String(), time.Tuesday.String(), time.Wednesday.String(),
		time.Thursday.String(), time.Friday.String(), time.Saturday.String(),
	}
	dayReg := strings.Join(allowedDays, "|")
	matches := regexp.MustCompile(`(` + dayReg + `) (\d{2}):(\d{2})`).FindStringSubmatch(m.Payload)

	wd, hour, min := 0, 19, 0
	var err error
	if matches == nil || len(matches) != 4 {
		formatError(m)
		return
	}

	for i, d := range allowedDays {
		if d == matches[1] {
			wd = i
		}
	}

	if err != nil {
		formatError(m)
		return
	}

	hour, err = strconv.Atoi(matches[2])
	if err != nil {
		formatError(m)
		return
	}
	min, err = strconv.Atoi(matches[3])
	if err != nil {
		formatError(m)
		return
	}
	bot.Config.Weekday = wd
	bot.Config.Hour = hour
	bot.Config.Minute = min

	bot.UpdateVote()
	bot.Send(m.Sender, "👌")
}

func handleCreate(m *tb.Message) {
	allowedUsers := make(map[string]bool, len(bot.Config.Admins))
	for _, admin := range bot.Config.Admins {
		allowedUsers[admin] = true
	}

	sender := strings.ToLower(m.Sender.Username)
	if _, ok := allowedUsers[sender]; !ok {
		bot.Send(m.Sender, "Permission denied, Пёс")
		return
	}

	log.Println("Force unpin message")

	err := bot.UnpinMessage()
	if err != nil {
		log.Printf("cannot upin, err: %v", err)
	}

	log.Println("Force create vote")
	if err := bot.CreateVote(); err != nil {
		log.Printf("caught error: %s", err)
	}
}

func formatError(m *tb.Message) {
	bot.Send(m.Sender, fmt.Sprintf("Лоховской формат: %s, нада типо Sunday 19:00", m.Payload))
}
