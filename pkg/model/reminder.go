package model

import "github.com/jinzhu/gorm"

// Reminder - model that represents remiders fot vote.
type Reminder struct {
	gorm.Model

	ReminderID int `json:"reminder_id"`
	VoteID     int `json:"vote_id"`
	Vote       Vote
}

// GetReminderByVoteID - retrieves Reminder for passed voteID.
func GetReminderByVoteID(voteID int) *Reminder {
	var rem *Reminder

	db.Take(rem, voteID)

	return rem
}

// CreateReminder - creates new reminder for passed voteID.
func CreateReminder(voteID int) *Reminder {
	var reminder = &Reminder{VoteID: voteID}

	db.Create(reminder)

	return reminder
}
