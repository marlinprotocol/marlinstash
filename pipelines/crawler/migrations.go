package crawler

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
var crawlerMigrations [4]*pipelines.Migration = [4]*pipelines.Migration{
	// 1 - Create Cluster table
	{
		Forward: func(db *pg.Tx) error {
			type Cluster struct {
				Host      string    `pg:",notnull,unique:dedup"`
				Inode     uint64    `pg:",notnull,unique:dedup"`
				Offset    uint64    `pg:",notnull,unique:dedup"`
				Ts        time.Time `pg:",notnull"`
				Level     string    `pg:",notnull"`
				Location  string    `pg:",notnull"`
				Ip        string    `pg:",notnull"`
				Cluster   string    `pg:",notnull"`
			}
			err := db.Model(&Cluster{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type Cluster struct{}
			err := db.Model(&Cluster{}).DropTable(nil)
			return err
		},
	},
	// 2 - Create ArchivedCluster table
	{
		Forward: func(db *pg.Tx) error {
			type ArchivedCluster struct {
				Host      string
				Inode     uint64
				Offset    uint64
				Ts        time.Time
				Level     string
				Location  string
				Ip        string
				Cluster   string
			}
			err := db.Model(&ArchivedCluster{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type ArchivedCluster struct{}
			err := db.Model(&ArchivedCluster{}).DropTable(nil)
			return err
		},
	},
	// 3 - Create Relay table
	{
		Forward: func(db *pg.Tx) error {
			type Relay struct {
				Host      string    `pg:",notnull,unique:dedup"`
				Inode     uint64    `pg:",notnull,unique:dedup"`
				Offset    uint64    `pg:",notnull,unique:dedup"`
				Ts        time.Time `pg:",notnull"`
				Level     string    `pg:",notnull"`
				Location  string    `pg:",notnull"`
				Ip        string    `pg:",notnull"`
				Cluster   string    `pg:",notnull"`
			}
			err := db.Model(&Relay{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type Relay struct{}
			err := db.Model(&Relay{}).DropTable(nil)
			return err
		},
	},
	// 4 - Create ArchivedRelay table
	{
		Forward: func(db *pg.Tx) error {
			type ArchivedRelay struct {
				Host      string
				Inode     uint64
				Offset    uint64
				Ts        time.Time
				Level     string
				Location  string
				Ip        string
				Cluster   string
			}
			err := db.Model(&ArchivedRelay{}).CreateTable(nil)
			return err
		},
		Backward: func(db *pg.Tx) error {
			type ArchivedRelay struct{}
			err := db.Model(&ArchivedRelay{}).DropTable(nil)
			return err
		},
	},
}
