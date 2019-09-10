package vote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/k33nice/vote-bot/pkg/model"
	"github.com/pkg/errors"
	tb "gopkg.in/tucnak/telebot.v2"
)

const userSymbol = "👤"
const ballSymbol = "⚽️"
const shitSymbol = "💩"

// HelpMessage - show help message.
const HelpMessage = `
	/help   - show this help message.
	/start  - start bot.
	/where  - show place where we playing.
	/info   - show info.
	/result - last vote result.
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

	b.Vote = &Vote{Format: b.Config.Formats.VoteFormat, RandAppeal: b.Config.Appeals[i]}
	b.Unique = getRandInt()

	msg, mrk, parseMode := b.getVoteMessage()
	m, err := b.Send(b.Channel, msg, mrk, parseMode)
	if err != nil {
		return errors.Wrap(err, "cannot send message to channel during a vote creation")
	}

	b.UnpinMessage()

	err = b.PinMessage(m)
	if err != nil {
		return errors.Wrap(err, "cannot pin message")
	}

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

		model.CreateVote(&model.Vote{
			VoteID:     b.getMsgID(),
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
		return pm, errors.Wrap(err, "cannot get pinned message")
	}

	if err = json.Unmarshal(r, &response); err != nil {
		return pm, errors.Wrap(err, "cannot Unmarshal response")
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
		b.Vote = &Vote{Format: b.Config.Formats.ResultFormat, RandAppeal: b.Config.Appeals[i]}
	}

	newMsg, mkp, parseMode := b.getVoteMessage()
	b.Edit(b.Pinned, newMsg, mkp, parseMode)

	return nil
}

// GetVoteResult - return result of current vote.
func (b *Bot) GetVoteResult() string {
	result := b.Config.NoResult
	if b.Pinned != nil {
		yesBtn, _ := b.getButtons()

		results := model.GetVoteResult(b.getMsgID())

		agree := 0
		disagree := 0
		for _, r := range results {
			if r.PressedBtn == yesBtn.Data {
				agree = r.Count
			} else {
				disagree = r.Count
			}
		}

		t := template.Must(template.New("").Parse(b.Config.Formats.ResultFormat))
		data := map[string]int{
			"Agree":    agree,
			"Disagree": disagree,
		}
		result = execTpl(t, data)
	}

	return result
}

// SendReminder - send reminder for players.
func (b *Bot) SendReminder() {
	if b.Pinned != nil {
		rem := model.GetReminderByVoteID(b.getMsgID())

		if rem != nil {
			return
		}

		voters := model.GetVotesByVoteID(b.getMsgID())
		yesBtn, _ := b.getButtons()

		var users []string
		for _, voter := range voters {
			if voter.PressedBtn == yesBtn.Data {
				users = append(users, fmt.Sprintf("[%s %s](tg://user?id=%d)", voter.FirstName, voter.LastName, voter.UserID))
			}
		}

		if len(users) >= 8 {
			t := template.Must(template.New("").Parse(b.Config.Formats.RemindFormat))
			data := map[string]string{
				"Appeal": b.Vote.RandAppeal,
				"Users":  strings.Join(users, " "),
			}
			result := execTpl(t, data)

			b.Send(b.Channel, result, tb.ModeMarkdown)
		}
	}
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
		voters := model.GetVotesByVoteID(b.getMsgID())
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

	t := template.Must(template.New("").Parse(b.Config.Formats.VoteFormat))

	data := map[string]interface{}{
		"Symbols":       strings.Repeat(userSymbol, agCount),
		"Appeal":        b.Vote.RandAppeal,
		"Yes":           yesBtn.Text,
		"Agree":         agCount,
		"AgreeNames":    agree,
		"No":            noBtn.Text,
		"Disagree":      dgCount,
		"DisagreeNames": disagree,
		"Date":          getDate(b.Config.Weekday, b.Config.Hour, b.Config.Minute).Format("2006-01-02 15:04"),
	}

	return execTpl(t, data)
}

func (b *Bot) getMsgID() int {
	msgID, _ := b.Pinned.MessageSig()
	id, _ := strconv.Atoi(msgID)

	return id
}

func getDate(wd, hour, minute int) time.Time {
	date := time.Now()

	weekday := int(date.Weekday())
	loc, _ := time.LoadLocation("Europe/Kiev")

	magicNum := 7
	if wd != int(time.Sunday) {
		magicNum = wd
	}

	diff := (magicNum - weekday)
	if diff < 0 {
		diff += 7
	}

	if weekday != 0 {
		date = date.AddDate(0, 0, diff)
	}

	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, loc)
}

func execTpl(t *template.Template, data interface{}) string {
	buf := bytes.Buffer{}
	t.Execute(&buf, data)
	return buf.String()
}
