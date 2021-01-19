package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type EntryLine struct {
	Service string
	Host    string
	Inode   string
	Offset  string
	Message string
}

type Config struct {
	Host     string
	Port     string
	DBName   string
	User     string
	Password string
}

type Worker struct {
	config *Config

	Entries chan *EntryLine
}

func CreateWorker(config *Config) *Worker {
	return &Worker{config, make(chan *EntryLine, 100)}
}

func (w *Worker) Run() {
	log.Info("Starting DB worker")

	timerWait := time.Second

	for {
		log.Info("Connecting to DB...")
		db, err := w.connect()
		if err != nil {
			log.Error("Connection error: ", err)

			// Exponential backoff
			time.Sleep(timerWait)
			timerWait *= 2
			if timerWait > 300*time.Second { // Cap backoff at 300s
				timerWait = 300 * time.Second
			}

			continue
		}
		defer db.Close()

		// TODO: Receive from channel and handle
	}
}

func (w *Worker) connect() (*sql.DB, error) {
	connstr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s", w.config.Host, w.config.Port, w.config.User, w.config.Password, w.config.DBName)

	db, err := sql.Open("postgres", connstr)
	return db, err
}
