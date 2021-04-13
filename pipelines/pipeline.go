package pipelines

import (
	"marlinstash/types"

	"github.com/go-pg/pg/v10"
)

type Pipeline interface {
	Setup(db *pg.DB) error
	ProcessEntry(db *pg.DB, entry *types.EntryLine) error
	Archive(tx *pg.Tx, service string, host string, inode uint64) error
}
