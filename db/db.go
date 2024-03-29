package db

import (
	"context"
	"fmt"
	"marlinstash/pipelines"
	"marlinstash/pipelines/feeder"
	"marlinstash/pipelines/probe"
	"marlinstash/types"
	"sync"
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
	ResetOffsetReqs chan *types.InodeOffsetReq
	Done            chan bool
}

func CreateWorker(config *pg.Options, done chan bool) *Worker {
	return &Worker{
		config,
		nil,
		make(map[string]pipelines.Pipeline),
		make(chan *types.EntryLine, 100),
		make(chan *types.InodeOffsetReq, 100),
		make(chan *types.InodeOffsetReq, 100),
		done,
	}
}

func (w *Worker) Run() {
	log.Info("Starting DB worker")

	tp := throughPutData{
		toDB: make(map[string]uint32),
		mu:   sync.Mutex{},
	}
	go tp.presentThroughput(5)

	timerWait := time.Second

	for {
		log.Info("Connecting to DB...")
		db := pg.Connect(w.config)
		db.AddQueryHook(dbLogger{})
		defer db.Close()

		err := w.setup(db)
		if err != nil {
			log.Error("Setup error: ", err)
			tp.putInfo("DBSetup-", 1)
			goto eb
		}
		tp.putInfo("DBSetup+", 1)

		// Process any hot entries first
		if w.hotEntry != nil {
			err := w.processEntry(db, w.hotEntry)
			if err != nil {
				log.Error("Insert error: ", err)
				tp.putInfo(w.hotEntry.Service+"_DBHotE-", 1)
				goto eb
			}
			tp.putInfo(w.hotEntry.Service+"_DBHotE-", 1)
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
					tp.putInfo(w.hotEntry.Service+"_DBHotE-", 1)
					goto eb
				}
				tp.putInfo(w.hotEntry.Service+"_DBHotE+", 1)
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
					tp.putInfo(req.Service+"_Ioffreq-", 1)
					goto eb
				}
				tp.putInfo(req.Service+"_Ioffreq+", 1)
				req.Resp <- inodeOffset
			case req, ok := <-w.ResetOffsetReqs:
				if !ok {
					log.Error("DB: Reset channel closed")
					close(w.Done)
					goto end
				}
				log.Info("Resetting: ", req.Service, req.Host, req.Inode)
				inodeOffset := &types.InodeOffset{
					Service: req.Service,
					Host:    req.Host,
					Inode:   req.Inode,
					Offset:  0,
				}

				tx, err := db.Begin()
				if err != nil {
					close(req.Resp)
					log.Error("Reset query error: ", err)
					goto eb
				}
				defer tx.Close()

				archivedEntryLine := &types.ArchivedEntryLine{
					Service: req.Service,
					Host:    req.Host,
					Inode:   req.Inode,
					Offset:  0,
					Message: "",
				}
				_, err = tx.Query(archivedEntryLine, "INSERT INTO archived_entry_lines SELECT * FROM entry_lines WHERE service = ? AND host = ? AND inode = ?;", req.Service, req.Host, req.Inode)
				if err != nil {
					tx.Rollback()
					close(req.Resp)
					log.Error("Reset query error: ", err)
					goto eb
				}

				_, err = tx.Query(archivedEntryLine, "DELETE FROM entry_lines WHERE service = ? AND host = ? AND inode = ?;", req.Service, req.Host, req.Inode)
				if err != nil {
					tx.Rollback()
					close(req.Resp)
					log.Error("Reset query error: ", err)
					goto eb
				}

				for _, pipeline := range w.pipelines {
					err := pipeline.Archive(tx, req.Service, req.Host, req.Inode)
					if err != nil {
						tx.Rollback()
						close(req.Resp)
						log.Error("Reset query error: ", err)
						goto eb
					}
				}

				_, err = db.Model(inodeOffset).OnConflict("(service, host, inode) DO UPDATE SET \"offset\" = 0").Insert()
				if err != nil {
					tx.Rollback()
					close(req.Resp)
					log.Error("Offset query error: ", err)
					goto eb
				}

				err = tx.Commit()
				if err != nil {
					close(req.Resp)
					log.Error("Reset query error: ", err)
					goto eb
				}

				req.Resp <- inodeOffset
			}
		}

		continue // End of loop, only specific control blocks below
	eb:
		// Exponential backoff
		tp.putInfo("DBEb+", 1)
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

	err = db.Model(&types.ArchivedEntryLine{}).CreateTable(&orm.CreateTableOptions{IfNotExists: true})
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
	w.pipelines["feeder"] = feeder.NewPipeline()

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

type throughPutData struct {
	toDB map[string]uint32
	mu   sync.Mutex
}

func (t *throughPutData) putInfo(key string, count uint32) {
	t.mu.Lock()
	t.toDB[key] = t.toDB[key] + count
	t.mu.Unlock()
}

func (t *throughPutData) presentThroughput(sec time.Duration) {
	for {
		time.Sleep(sec * time.Second)
		t.mu.Lock()
		log.Info(fmt.Sprintf("[Stash stats] To DB %v", t.toDB))
		t.toDB = make(map[string]uint32)
		t.mu.Unlock()
	}
}
