package rest

import (

)

type Message struct {
	Id int64			`json:"id" db:"id"`
	Date string			`json:"date" db:"date"`
	Message string			`json:"message" db:"message"`
	IsDeleted bool			`json:"isDeleted" db:"status_is_deleted"`
	User interface{}		`json:"user" db:"user"`
	Forum interface{}		`json:"forum" db:"forum"`

	Likes		int	`json:"likes" db:"calc_rate_positive"`
	Dislikes	int	`json:"dislikes" db:"calc_rate_negative"`
	Points		int	`json:"points" db:"calc_rate_points"`

}
/*
type Rate struct {
	Message int64	`json:"message" db:"message_id"`
	IsLike	bool	`json:"isLike" db:"status_is_rate_like"`
}

func (m *Message) InsertIntoDb(db *sqlx.DB) (sql.Result, error) {
	log.Printf("[ L ] Inserting to Message table entry:\r\n %+v", m)
	if m == nil {
		return nil, errors.New("Error: Message struct is nil")
	}
	return db.Exec(
		"INSERT INTO Message (forum, user, date, status_is_deleted, message) VALUES ( ? , ? , ? , ? , ? )",
			m.Forum.(string), m.User.(string), m.Date, m.IsDeleted, m.Message);
}

func MessageSetDeletedById(id int64, db *sqlx.DB, deleted bool ) (error) {
	_, err := db.Exec("UPDATE Message SET status_is_deleted = ? WHERE id = ?",  deleted, id)
	return err
}


func getMessageRatesById( id int64, db *sqlx.DB  ) ([]Rate, error ) {
	var rates []Rate;
	err := db.Select(&rates, "SELECT status_is_rate_like FROM UserMessageRate WHERE message_id = ?", id)
	return rates, err
}

func getMessagePointsById( id int64, m *Message , db *sqlx.DB) {
	rates, _ := getMessageRatesById(id, db)
	for _, rate := range rates {
		if rate.IsLike { m.Likes += 1 } else { m.Dislikes += 1}
	}
	m.Points = m.Likes - m.Dislikes
}

func getMessageById( id int64 ,  db *sqlx.DB) (*Message, error) {
	msg  := new(Message)
	err := db.Get(msg, "SELECT * FROM Message WHERE id = ?", id)
	if msg.Forum != nil { 	msg.Forum = string(msg.Forum.([]uint8)) }
	if msg.User != nil { msg.User = string(msg.User.([]uint8)) }
	// get likes
	getMessagePointsById(id, msg, db)
	return msg, err
}

func (m * Message) getPoints(db *sqlx.DB) {
	getMessagePointsById(m.Id, m, db)

}

func voteOnMessageById( id int64, is_like bool, db *sqlx.DB) (error) {
	_, err := db.Exec("INSERT INTO UserMessageRate (message_id, status_is_rate_like) VALUES(?, ?) ", id, is_like)
	return err
}


 */