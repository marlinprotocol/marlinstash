package spamcheck

import (
	"marlinstash/pipelines"
	"marlinstash/pipelines/parsers"
	"marlinstash/types"
	"regexp"
	"time"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
)

type Pipeline struct {
	spdlog        *parsers.SpdlogParser
	spamcheckParser *regexp.Regexp
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		parsers.NewSpdlogParser(),
		regexp.MustCompile("Spam checked message: ([0-9]+)"),
	}
}

type Spamcheck struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	MessageId string
}

type ArchivedSpamcheck struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	MessageId string
}


func (p *Pipeline) Setup(db *pg.DB) error {
	return pipelines.ApplyMigrations(db, "spamcheck", spamcheckMigrations[:], uint64(len(spamcheckMigrations)))
}

func (p *Pipeline) ProcessEntry(db *pg.DB, entry *types.EntryLine) error {
	ts, level, location, msg, err := p.spdlog.Parse(entry.Message)
	if err != nil {
		// Ignore parse error
		log.Debug("Dropped entry: ", entry.Message, err)
		return nil
	}

	parts := p.spamcheckParser.FindStringSubmatch(msg)
	if len(parts) == 2 {
		parts = parts[1:]
		if err != nil {
			// Ignore parse error
			log.Debug("Dropped entry: ", msg, err)
			return nil
		}
		obj := &Spamcheck{
			entry.Host,
			entry.Inode,
			entry.Offset,
			ts,
			level,
			location,
			parts[0],
		}
		_, err = db.Model(obj).
			OnConflict("DO NOTHING").
			Insert()
		if err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (p *Pipeline) Archive(tx *pg.Tx, service string, host string, inode uint64) error {
	_, err := tx.Query(&ArchivedSpamcheck{}, "INSERT INTO archived_spamchecks SELECT * FROM spamchecks WHERE host = ? AND inode = ?;", host, inode)
	if err != nil {
		return err
	}

	_, err = tx.Query(&ArchivedSpamcheck{}, "DELETE FROM spamchecks WHERE host = ? AND inode = ?;", host, inode)
	if err != nil {
		return err
	}

	return nil
}
