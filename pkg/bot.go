package vote

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/k33nice/vote-bot/pkg/model"
	tb "gopkg.in/tucnak/telebot.v2"
)

const userSymbol = "👤"
const ballSymbol = "⚽️"
const shitSymbol = "💩"

// HelpMessage - show help message.
const HelpMessage = `
	/help show this help message.
	/start start bot.
	/where show place where we playing.
	/info show info.
`

// Bot - represent a separate telegram bot instance.
type Bot struct {
	tb.Bot

	Unique  int
	Config  *Config
	Vote    *Vote
	Pinned  tb.Editable
	Channel *tb.Chat
}

// NewBot - return new Bot instance.
func NewBot(s tb.Settings, config *Config) (*Bot, error) {
	b, err := tb.NewBot(s)

	if err != nil {
		return nil, err
	}
	return &Bot{Bot: *b, Config: config, Unique: getRandInt()}, nil
}

func getRandInt() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Int()
}

var format = `
	%s
	*Шо вы %s?*

	_%s (%d)_
	%s
	----------
	_%s (%d)_
	%s
`

// Vote - struct for vote.
type Vote struct {
	Format     string
	RandAppeal string
}

func (b *Bot) getButtons() (*tb.InlineButton, *tb.InlineButton) {
	yesBtn := tb.InlineButton{
		Unique: fmt.Sprintf("yes_%d", b.Unique),
		Text:   "Да",
		Data:   "1",
	}

	noBtn := tb.InlineButton{
		Unique: fmt.Sprintf("no_%d", b.Unique),
		Text:   "Нет",
		Data:   "0",
	}

	return &yesBtn, &noBtn
}

// CreateVote - creating new message for voting.
func (b *Bot) CreateVote() error {
	if b.Channel == nil {
		return errors.New("No channel")
	}

	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(len(b.Config.Appeals))

	b.Vote = &Vote{Format: format, RandAppeal: b.Config.Appeals[i]}
	b.Unique = getRandInt()

	msg, mrk, parseMode := b.getVoteMessage()
	m, err := b.Send(b.Channel, msg, mrk, parseMode)
	if err != nil {
		log.Printf("err = %+v\n", err)
		return err
	}

	b.PinMessage(m)

	return nil
}

// CreateHandlers - create handlers for buttons.
func (b *Bot) CreateHandlers() {
	yB, nB := b.getButtons()
	b.Handle(yB, b.buttonHandler(*yB))

	b.Handle(nB, b.buttonHandler(*nB))
}

func (b *Bot) buttonHandler(btn tb.InlineButton) func(*tb.Callback) {
	return func(c *tb.Callback) {
		b.Respond(c, &tb.CallbackResponse{Text: btn.Text})

		msgID, _ := b.Pinned.MessageSig()
		id, err := strconv.Atoi(msgID)
		if err != nil {
			log.Panic(err)
		}

		model.CreateVote(&model.Vote{
			VoteID:     id,
			UserID:     c.Sender.ID,
			VoterName:  c.Sender.Username,
			FirstName:  c.Sender.FirstName,
			LastName:   c.Sender.LastName,
			PressedBtn: c.Data,
		})

		newMsg, mkp, parseMode := b.getVoteMessage()
		b.Edit(b.Pinned, newMsg, mkp, parseMode)
	}
}

// PinnedMessage - raw telegram api pinned message.
type PinnedMessage struct {
	ID   float64 `json:"message_id"`
	Date float64
	Text string
	From struct {
		Username string
		IsBot    bool `json:"is_bot"`
	}
	Chat struct {
		ID float64
	}
}

// MessageSig - to implement telebot.Edditable
func (pm *PinnedMessage) MessageSig() (string, int64) {
	return strconv.Itoa(int(pm.ID)), int64(pm.Chat.ID)
}

// GetPinnedMessage - return pinned message in chat by id.
func (b *Bot) GetPinnedMessage(chatID int) (*PinnedMessage, error) {
	var pm *PinnedMessage

	var response struct {
		Ok     bool
		Result struct {
			PinnedMessage PinnedMessage `json:"pinned_message"`
		}
	}

	r, err := b.Raw("getChat", map[string]string{"chat_id": strconv.Itoa(chatID)})
	if err != nil {
		return pm, err
	}

	if err = json.Unmarshal(r, &response); err != nil {
		return pm, err
	}

	if response.Result.PinnedMessage.ID > 0 {
		pm = &response.Result.PinnedMessage
	}

	return pm, nil
}

// PinMessage - pin message in chat.
func (b *Bot) PinMessage(m *tb.Message) error {
	if b.Pinned != nil {
		log.Print("Pinnded exists")
		return nil
	}
	if err := b.Pin(m); err != nil {
		log.Print(err)
		return err
	}
	b.Pinned = m

	return nil
}

// UnpinMessage - unpin message in current chat.
func (b *Bot) UnpinMessage() error {
	if b.Channel == nil {
		return errors.New("Not in channel")
	}

	err := b.Unpin(b.Channel)
	if err != nil {
		return err
	}
	b.Pinned = nil

	return nil
}

// UpdateVote - update vote.
func (b *Bot) UpdateVote() error {
	if b.Vote == nil {
		rand.Seed(time.Now().UnixNano())
		i := rand.Intn(len(b.Config.Appeals))
		b.Vote = &Vote{Format: format, RandAppeal: b.Config.Appeals[i]}
	}

	newMsg, mkp, parseMode := b.getVoteMessage()
	b.Edit(b.Pinned, newMsg, mkp, parseMode)

	return nil
}

func (b *Bot) getVoteMessage() (string, *tb.ReplyMarkup, tb.ParseMode) {
	yB, nB := b.getButtons()
	inlineKeys := [][]tb.InlineButton{
		[]tb.InlineButton{*yB},
		[]tb.InlineButton{*nB},
	}

	return b.voteCaption(), &tb.ReplyMarkup{InlineKeyboard: inlineKeys}, tb.ModeMarkdown
}

func (b *Bot) voteCaption() string {
	var agree string
	var disagree string
	agCount := 0
	dgCount := 0

	yesBtn, noBtn := b.getButtons()
	if b.Pinned != nil {
		msgID, _ := b.Pinned.MessageSig()
		id, _ := strconv.Atoi(msgID)
		voters := model.GetVotesByVoteID(id)
		for _, voter := range voters {

			var symbol string
			if voter.PressedBtn == yesBtn.Data {
				agCount++
				symbol = ballSymbol
			} else {
				dgCount++
				symbol = shitSymbol
			}

			str := fmt.Sprintf(
				"\n %s [%s %s](tg://user?id=%d)",
				symbol,
				voter.FirstName,
				voter.LastName,
				voter.UserID,
			)

			if voter.PressedBtn == yesBtn.Data {
				agree += str
			} else {
				disagree += str
			}
		}
	}

	return fmt.Sprintf(
		b.Vote.Format,
		strings.Repeat(userSymbol, agCount),
		b.Vote.RandAppeal,
		yesBtn.Text,
		agCount,
		agree,
		noBtn.Text,
		dgCount,
		disagree,
	)
}
