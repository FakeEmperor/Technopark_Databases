package rest

import (
	"gopkg.in/gorp.v1"
	"database/sql"
)

type Message struct {
	Id int				`json:"id" db:"id"`
	Date string			`json:"date" db:"date"`
	Message string			`json:"message" db:"message"`
	IsDeleted bool			`json:"isDeleted" db:"state_is_deleted"`
	User interface{}		`json:"user" db:"user"`
	Forum interface{}		`json:"forum" db:"forum_id"`

	Likes		int	`json:"likes"`
	Dislikes	int	`json:"dislikes"`
	Points		int	`json:"points"`

}

type Rate struct {
	Message int	`json:"message" db:"message_id"`
	User	string	`json:"user" db:"user"`
	IsLike	bool	`json:"isLike" db:"status_is_rate_like"`
}

func (m *Message) InsertIntoDb(db *gorp.DbMap) (sql.Result, error) {
	return db.Exec(
		"INSERT INTO Message (forum_id, user, date, state_is_deleted, message) VALUES ( ?, ?, ?, ?, ?)",
			m.Forum, m.User, m.Date, m.IsDeleted, m.Message);
}

func MessageSetDeletedById(id int, db *gorp.DbMap, deleted bool ) (error) {
	_, err := db.Exec("UPDATE Message SET status_is_deleted = ? WHERE id = ?", id, deleted)
	return err
}


func getMessageRatesById( id int, db *gorp.DbMap, include_users bool  ) ([]Rate, error ) {
	var rates []Rate;
	selecting := "status_is_rate_like"
	if include_users == true { selecting += ", user" }
	_, err := db.Select(&rates, "SELECT "+selecting+" FROM UserMessageRate WHERE message_id = ?", id)
	return rates, err
}

func getMessageById( id int ,  db *gorp.DbMap) (*Message, error) {
	msg  := new(Message)
	err := db.SelectOne(msg, "SELECT * FROM Message WHERE id = ?", id)
	// get likes
	rates, _ := getMessageRatesById(id, db, false)
	for _, rate := range rates {
		if rate.IsLike { msg.Likes += 1 } else { msg.Dislikes += 1}
	}
	msg.Points = msg.Likes - msg.Dislikes
	return msg, err
}

func voteOnMessageById( id int, user string, is_like bool, db *gorp.DbMap) (error) {
	_, err := db.Exec("INSERT INTO UserMssageRate (user, message_id, status_is_rate_like) VALUES(?, ?, ?) "+
	"ON DUPLICATE KEY UPDATE status_is_rate_like = ?", user, id, is_like, is_like)
	return err
}

