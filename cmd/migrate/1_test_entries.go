package main

import (
	"github.com/go-pg/migrations/v8"
)

type EntryLine struct {
	Service string `pg:",unique:dedup"`
	Host    string `pg:",unique:dedup"`
	Inode   uint64 `pg:",unique:dedup"`
	Offset  uint64 `pg:",unique:dedup"`
	Message string
}

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		err := db.Model(&EntryLine{}).CreateTable(nil)
		return err
	}, func(db migrations.DB) error {
		err := db.Model(&EntryLine{}).DropTable(nil)
		return err
	})
}
