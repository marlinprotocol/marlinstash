package feeder

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
var feederMigrations [2]*pipelines.Migration = [2]*pipelines.Migration{
	// 1 - Create MsgSend table
	{
		Forward: func(db *pg.Tx) error {
			type MsgSend struct {
				Host      string    `pg:",notnull,unique:dedup"`
				Inode     uint64    `pg:",notnull,unique:dedup"`
				Offset    uint64    `pg:",notnull,unique:dedup"`
				Ts        time.Time `pg:",notnull"`
				Level     string    `pg:",notnull"`
				Location  string    `pg:",notnull"`
				MessageId uint64    `pg:",notnull"`
				Cluster   string    `pg:",notnull"`
			}
			err := db.Model(&MsgSend{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type MsgSend struct{}
			err := db.Model(&MsgSend{}).DropTable(nil)
			return err
		},
	},
	// 2 - Create ArchivedMsgSend table
	{
		Forward: func(db *pg.Tx) error {
			type ArchivedMsgSend struct {
				Host      string
				Inode     uint64
				Offset    uint64
				Ts        time.Time
				Level     string
				Location  string
				MessageId uint64
				Cluster   string
			}
			err := db.Model(&ArchivedMsgSend{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type ArchivedMsgSend struct{}
			err := db.Model(&ArchivedMsgSend{}).DropTable(nil)
			return err
		},
	},
}
