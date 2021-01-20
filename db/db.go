package db

import (
	"time"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
)

type EntryLine struct {
	Service string `pg:",unique:dedup"`
	Host    string `pg:",unique:dedup"`
	Inode   int64  `pg:",unique:dedup"`
	Offset  int64  `pg:",unique:dedup"`
	Message string
}

type Worker struct {
	config   *pg.Options
	hotEntry *EntryLine

	Entries chan *EntryLine
	Done    chan *EntryLine
}

func CreateWorker(config *pg.Options) *Worker {
	return &Worker{config, nil, make(chan *EntryLine, 100), make(chan *EntryLine, 100)}
}

func (w *Worker) Run() {
	log.Info("Starting DB worker")

	timerWait := time.Second

	for {
		log.Info("Connecting to DB...")
		db := pg.Connect(w.config)
		defer db.Close()

		// Process any hot entries first
		if w.hotEntry != nil {
			err := w.processEntry(db, w.hotEntry)
			if err != nil {
				log.Error("Insert error: ", err)
				goto eb
			}
			w.Done <- w.hotEntry
			w.hotEntry = nil
		}

		// Listen for new entries
		for {
			entry, ok := <-w.Entries
			if !ok {
				log.Error("DB: Entries channel closed")
				close(w.Done)
				goto end
			}
			w.hotEntry = entry
			err := w.processEntry(db, w.hotEntry)
			if err != nil {
				log.Error("Insert error: ", err)
				goto eb
			}
			w.Done <- w.hotEntry
			w.hotEntry = nil
		}

		// End of loop, only specific control blocks below
	eb:
		// Exponential backoff
		time.Sleep(timerWait)
		timerWait *= 2
		if timerWait > 300*time.Second { // Cap backoff at 300s
			timerWait = 300 * time.Second
		}

		continue
	}
	end:
}

func (w *Worker) processEntry(db *pg.DB, entry *EntryLine) error {
	_, err := db.Model(entry).OnConflict("DO NOTHING").Insert()
	return err
}

