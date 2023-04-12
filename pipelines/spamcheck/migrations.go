package spamcheck

import (
	"marlinstash/pipelines"
	"time"

	"github.com/go-pg/pg/v10"
)

//-------- HERE BE DRAGONS --------//
// DO NOT change or remove elements from this list
// Will cause the binary to desync with the DB and
// potentially corrupt it beyond recovery
// Adding elements is relatively safer (key word _relatively_)
// Can still blow up the db if you don't know what you're doing
var spamcheckMigrations [2]*pipelines.Migration = [2]*pipelines.Migration{
	// 1 - Create Spamcheck table
	{
		Forward: func(db *pg.Tx) error {
			type Spamcheck struct {
				Host      string    `pg:",notnull,unique:dedup"`
				Inode     uint64    `pg:",notnull,unique:dedup"`
				Offset    uint64    `pg:",notnull,unique:dedup"`
				Ts        time.Time `pg:",notnull"`
				Level     string    `pg:",notnull"`
				Location  string    `pg:",notnull"`
				MessageId string    `pg:",notnull"`
			}
			err := db.Model(&Spamcheck{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type Spamcheck struct{}
			err := db.Model(&Spamcheck{}).DropTable(nil)
			return err
		},
	},
	// 2 - Create ArchivedSpamcheck table
	{
		Forward: func(db *pg.Tx) error {
			type ArchivedSpamcheck struct {
				Host      string
				Inode     uint64
				Offset    uint64
				Ts        time.Time
				Level     string
				Location  string
				MessageId string
			}
			err := db.Model(&ArchivedSpamcheck{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type ArchivedSpamcheck struct{}
			err := db.Model(&ArchivedSpamcheck{}).DropTable(nil)
			return err
		},
	},
}
