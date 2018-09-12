package model

import (
	"github.com/jinzhu/gorm"
)

// Vote - model for user collection.
type Vote struct {
	gorm.Model

	VoteID     int    `json:"vote_id"`
	UserID     int    `json:"user_id"`
	VoterName  string `json:"voter_name"`
	FirstName  string `json:"voter_first_name"`
	LastName   string `json:"voter_last_name"`
	PressedBtn string `json:"pressed_btn"`
}

var db = GetEngine()

// GetVotes - return votes list.
func GetVotes(where ...string) []Vote {
	var votes []Vote

	if where != nil {
		db.Where(where).Find(&votes)
	} else {
		db.Find(&votes)
	}

	return votes
}

// GetVotesCount - return votes count for vote id.
func GetVotesCount(voteID int) int {
	var count int

	db.Model(&Vote{}).Where(Vote{VoteID: voteID}).Count(&count)

	return count
}

// GetVotesByVoteID - return votes by vote id.
func GetVotesByVoteID(voteID int) []Vote {
	var votes []Vote

	db.Where(Vote{VoteID: voteID}).Find(&votes)

	return votes
}

// GetVote - return vote by `id`.
func GetVote(id int) Vote {
	var vote Vote

	db.Take(&vote, id)

	return vote
}

// CreateVote - create new vote in database.
func CreateVote(v *Vote) Vote {
	var vote Vote
	db.Where(&Vote{VoteID: v.VoteID, UserID: v.UserID}).Assign(v).FirstOrCreate(&vote)

	return vote
}

// UpdateVote - create new vote in database.
func UpdateVote(id int, v Vote) {
	vote := GetVote(id)

	db.Model(&vote).Updates(v)
}
