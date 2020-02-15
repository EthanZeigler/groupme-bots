package adapter

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	srv "github.com/ethanzeigler/groupme/botserver"
	_ "github.com/lib/pq"
	"strconv"
	"time"
)

type SortType int

const (
	QuoteIDSort SortType = 0
	DateSort SortType = 1
	RandomSort SortType = 2
)

// Represents a Row inside of the quote db table
type Quote struct {
	ID *uint64
	Name *string
	Quote *string
	Date *time.Time
	GroupID *uint64
	SubmitterID *string
}


type MemeDB struct {
	// the database connection
	db *sql.DB
}

func NewMemeDB(conStr string) (*MemeDB, error) {
	var memeDB MemeDB
	var err error
	// open heroku db connection

	//db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	//connStr := "user=bots dbname=bots port=5437 host=127.0.0.1 connect_timeout=5"
	//connStr := "user=bots dbname=bots-debug port=5437 connect_timeout=5 sslmode=disable"
	//conStr := "user=bots dbname=bots connect_timeout=5"
	memeDB.db, err = sql.Open("postgres", conStr)
	if err != nil {
		return nil, err
	} else {
		return &memeDB, nil
	}
}


// gets a random quote from the given user
func (d *MemeDB) GetUserQuote(name string, callback srv.Callback) (quoteRow Quote, err error) {
	quotes, err := d.GetQuotes(name, callback, 1, RandomSort)

	if err != nil {
		return Quote{}, err
	}
	return quotes[0], err
}

func (d *MemeDB) WriteUserQuote(name string, quote string, callback srv.Callback) error {
	curr := time.Now()
	groupID, err := strconv.Atoi(callback.GroupID)
	if err != nil {
		return err
	}
	_, err = d.db.Exec("INSERT INTO quotes (name, quote, group_id, date, submit_by) VALUES ($1, $2, $3, to_date($4,'YYYY-MM-DD'), $5)",
		name, quote, groupID, fmt.Sprintf("%d-%02d-%02d\n",
			curr.Year(), curr.Month(), curr.Day()), callback.SenderID)
	if err != nil {
		return err
	}
	return nil
}

func (d *MemeDB) GetQuotes(name string, callback srv.Callback, limit int, sortType SortType) (quotes []Quote, err error) {
	groupID, err := strconv.Atoi(callback.GroupID)
	if err != nil {
		return make([]Quote, 0, 1), err
	}
	var rows *sql.Rows
	switch sortType {
	case DateSort:
		rows, err = d.db.Query("SELECT id, name, quote, group_id, date, submit_by FROM quotes " +
			"WHERE name LIKE $1 AND group_id=$2 ORDER BY date DESC LIMIT $3", name, groupID, limit)
		break
	case QuoteIDSort:
		rows, err = d.db.Query("SELECT id, name, quote, group_id, date, submit_by FROM quotes " +
			"WHERE name LIKE $1 AND group_id=$2 ORDER BY id DESC LIMIT $3", name, groupID, limit)
		break
	case RandomSort:
		rows, err = d.db.Query("SELECT id, name, quote, group_id, date, submit_by FROM quotes " +
			"WHERE name LIKE $1 AND group_id=$2 ORDER BY random() LIMIT $3", name, groupID, limit)
		break
	default:
		return make([]Quote, 0, 1), errors.New("illegal SortType")
	}

	if err != nil {
		return make([]Quote, 0, 1), err
	}
	var i = 0
	for ; rows.Next(); i++ {
		var name, quote, submitterID string
		var date time.Time
		var quoteID, groupID uint64

		err := rows.Scan(&quoteID, &name, &quote, &groupID, &date, &submitterID)
		if err != nil {
			return make([]Quote, 0, 1), err
		}
		quotes = append(quotes, Quote{
			Name: &name, Quote: &quote,
			Date: &date, GroupID: &groupID,
			ID:&quoteID, SubmitterID:&submitterID})
	}

	if len(quotes) == 0 {
		return make([]Quote, 0, 1), errors.New("no quotes found")
	}
	err = nil
	return
}

func (d *MemeDB) DeleteQuote(quote Quote) (sql.Result, error){
	return d.db.Exec("DELETE FROM quotes WHERE id=$1", quote.ID)
}

func (d *MemeDB) TestQuery(buffer *bytes.Buffer) error {
	rows, err := d.db.Query("SELECT * FROM quotes")
	if err != nil {
		return err
	}

	for i:= 0; rows.Next(); i++ {
		row, err := rows.Columns()
		if err != nil {
			buffer.Write([]byte("Cannot query db: "))
			buffer.Write([]byte(err.Error()+"\n"))
		} else {
			buffer.Write([]byte("Row name: "))
			for _, e := range row {
				buffer.Write([]byte(e+"\t"))
			}
			data, _ := rows.ColumnTypes()
			buffer.Write([]byte("\n"))
			buffer.Write([]byte("Row type: "))
			for _, e := range data {
				buffer.Write([]byte(e.ScanType().Name()+"\t"))
			}
			buffer.Write([]byte("\n"))

			var name, quote string
			var date time.Time

			err = rows.Scan(&name, &quote, &date)
			if err != nil {
				buffer.Write([]byte("Could not load data: "+err.Error()))
				return err
			}
			buffer.Write([]byte("Row Data: "+name+"\t"+quote+"\t"+date.String()+"\n"))
		}
		if i > 10 {
			buffer.Write([]byte("Stopping...\n"))
			break
		}
	}
	return nil
}

