package db

import (
	"context"
	"time"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
)

const retentionFirstRun = time.Minute
const retentionInterval = time.Hour

// Tables of raw log lines have no time of their own, so the time at which a row
// is stored is kept in stored_at once retention is enabled. Rows that exist at
// that point get the time at which the column is added.
// Column is added at the end of both tables, archival copies rows with SELECT *.
func setupRetention(db *pg.DB) error {
	return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		for _, query := range []string{
			"ALTER TABLE entry_lines ADD COLUMN IF NOT EXISTS stored_at timestamptz NOT NULL DEFAULT now()",
			"ALTER TABLE archived_entry_lines ADD COLUMN IF NOT EXISTS stored_at timestamptz NOT NULL DEFAULT now()",
			"CREATE INDEX IF NOT EXISTS entry_lines_stored_at_idx ON entry_lines USING brin (stored_at)",
		} {
			_, err := tx.Exec(query)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// applyRetention deletes everything older than retentionDays before now.
// Parsed tables are cut on the time in the log line, raw lines on the time they were stored.
func (w *Worker) applyRetention(db *pg.DB, now time.Time) error {
	if w.retentionDays <= 0 {
		return nil
	}
	olderThan := now.Add(-time.Duration(w.retentionDays) * 24 * time.Hour)
	deleted := 0

	for name, pipeline := range w.pipelines {
		n, err := pipeline.Prune(db, olderThan)
		deleted += n
		if err != nil {
			log.Error("Retention failed for pipeline ", name)
			return err
		}
	}

	for _, table := range []string{"entry_lines", "archived_entry_lines"} {
		res, err := db.Exec("DELETE FROM "+table+" WHERE stored_at < ?", olderThan)
		if err != nil {
			return err
		}
		deleted += res.RowsAffected()
	}

	log.Info("Retention: deleted ", deleted, " rows older than ", olderThan.UTC().Format(time.RFC3339))
	return nil
}
