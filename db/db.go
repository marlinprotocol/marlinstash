package db

import (
	"marlinstash/types"
	"time"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
)


type Worker struct {
	config   *pg.Options
	hotEntry *types.EntryLine

	Entries chan *types.EntryLine
	InodeOffsetReqs chan *types.InodeOffsetReq
	Done chan bool
}

func CreateWorker(config *pg.Options, done chan bool) *Worker {
	return &Worker{config, nil, make(chan *types.EntryLine, 100), make(chan *types.InodeOffsetReq, 100), done}
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
			w.hotEntry = nil
		}

		// Listen for new entries
		select {
		case entry, ok := <-w.Entries:
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
			w.hotEntry = nil
		case req, ok := <-w.InodeOffsetReqs:
			if !ok {
				log.Error("DB: Reqs channel closed")
				close(w.Done)
				goto end
			}
			inodeOffset := &types.InodeOffset{
				Service: req.Service,
				Host: req.Host,
				Inode: req.Inode,
				Offset: 0,
			}
			_, err := db.Model(inodeOffset).
					Where("service = ?", req.Service).
					Where("host = ?", req.Host).
					Where("inode = ?", req.Inode).
					SelectOrInsert()
			if err != nil {
				close(req.Resp)
			} else {
				req.Resp<- inodeOffset
			}
		}

		continue // End of loop, only specific control blocks below
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

func (w *Worker) processEntry(db *pg.DB, entry *types.EntryLine) error {
	_, err := db.Model(entry).OnConflict("DO NOTHING").Insert()

	inodeOffset := &types.InodeOffset{
		Service: entry.Service,
		Host: entry.Host,
		Inode: entry.Inode,
		Offset: entry.Offset,
	}
	_, err = db.Model(inodeOffset).OnConflict("DO UPDATE offset = greatest(offset, EXCLUDED.offset)").Insert()

	return err
}

