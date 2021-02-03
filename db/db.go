package db

import (
	"context"
	"marlinstash/pipelines"
	"marlinstash/pipelines/probe"
	"marlinstash/types"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	log "github.com/sirupsen/logrus"
)

type dbLogger struct{}

func (d dbLogger) BeforeQuery(c context.Context, q *pg.QueryEvent) (context.Context, error) {
	return c, nil
}

func (d dbLogger) AfterQuery(c context.Context, q *pg.QueryEvent) error {
	res, _ := q.FormattedQuery()
	log.Debug(string(res))
	return nil
}

type Worker struct {
	config    *pg.Options
	hotEntry  *types.EntryLine
	pipelines map[string]pipelines.Pipeline

	Entries         chan *types.EntryLine
	InodeOffsetReqs chan *types.InodeOffsetReq
	Done            chan bool
}

func CreateWorker(config *pg.Options, done chan bool) *Worker {
	return &Worker{config, nil, make(map[string]pipelines.Pipeline), make(chan *types.EntryLine, 100), make(chan *types.InodeOffsetReq, 100), done}
}

func (w *Worker) Run() {
	log.Info("Starting DB worker")

	timerWait := time.Second

	for {
		log.Info("Connecting to DB...")
		db := pg.Connect(w.config)
		db.AddQueryHook(dbLogger{})
		defer db.Close()

		err := w.setup(db)
		if err != nil {
			log.Error("Setup error: ", err)
			goto eb
		}

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
		for {
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
					Host:    req.Host,
					Inode:   req.Inode,
					Offset:  0,
				}
				_, err := db.Model(inodeOffset).
					Where("service = ?", req.Service).
					Where("host = ?", req.Host).
					Where("inode = ?", req.Inode).
					SelectOrInsert()
				if err != nil {
					close(req.Resp)
					log.Error("Offset query error: ", err)
					goto eb
				}
				req.Resp <- inodeOffset
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

func (w *Worker) setup(db *pg.DB) error {
	// Setup migrations table
	err := db.Model(&pipelines.MigrationState{}).CreateTable(&orm.CreateTableOptions{IfNotExists: true})
	if err != nil {
		return err
	}

	// Setup entry line table
	err = db.Model(&types.EntryLine{}).CreateTable(&orm.CreateTableOptions{IfNotExists: true})
	if err != nil {
		return err
	}

	// Setup inode offset table
	err = db.Model(&types.InodeOffset{}).CreateTable(&orm.CreateTableOptions{IfNotExists: true})
	if err != nil {
		return err
	}

	// register pipelines
	w.pipelines["probe"] = probe.NewPipeline()

	for _, pipeline := range w.pipelines {
		err := pipeline.Setup(db)
		if err != nil {
			return err
		}
	}

	return nil
}
 
func (w *Worker) processEntry(db *pg.DB, entry *types.EntryLine) error {
	_, err := db.Model(entry).OnConflict("DO NOTHING").Insert()

	if pipeline, ok := w.pipelines[entry.Service]; ok {
		err := pipeline.ProcessEntry(db, entry)
		if err != nil {
			return err
		}
	}

	inodeOffset := &types.InodeOffset{
		Service: entry.Service,
		Host:    entry.Host,
		Inode:   entry.Inode,
		Offset:  entry.Offset,
	}
	_, err = db.Model(inodeOffset).OnConflict("(service, host, inode) DO UPDATE SET \"offset\" = greatest(inode_offset.offset, EXCLUDED.offset)").Insert()

	return err
}
