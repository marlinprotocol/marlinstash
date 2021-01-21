package main

import (
	"github.com/go-pg/migrations/v8"
)

type InodeOffset struct {
	Service string `pg:",unique:dedup"`
	Host    string `pg:",unique:dedup"`
	Inode   uint64 `pg:",unique:dedup"`
	Offset  uint64
}

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		err := db.Model(&InodeOffset{}).CreateTable(nil)
		return err
	}, func(db migrations.DB) error {
		err := db.Model(&InodeOffset{}).DropTable(nil)
		return err
	})
}
