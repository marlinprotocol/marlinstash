package pipelines

import (
	"marlinstash/types"
	"time"

	"github.com/go-pg/pg/v10"
)

type Pipeline interface {
	Setup(db *pg.DB) error
	ProcessEntry(db *pg.DB, entry *types.EntryLine) error
	Archive(tx *pg.Tx, service string, host string, inode uint64) error
	// Prune deletes rows with a log time before olderThan, returns the number of rows deleted
	Prune(db *pg.DB, olderThan time.Time) (int, error)
}
