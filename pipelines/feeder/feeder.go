package feeder

import (
	"marlinstash/pipelines"
	"marlinstash/pipelines/parsers"
	"marlinstash/types"
	"regexp"
	"strconv"
	"time"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
)

type Pipeline struct {
	spdlog        *parsers.SpdlogParser
	msgSendParser *regexp.Regexp
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		parsers.NewSpdlogParser(),
		regexp.MustCompile("Sending message ([0-9]+) to (\\w+)"),
	}
}

type MsgSend struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	MessageId uint64
	Cluster   string
}

func (p *Pipeline) Setup(db *pg.DB) error {
	return pipelines.ApplyMigrations(db, "feeder", feederMigrations[:], uint64(len(feederMigrations)))
}

func (p *Pipeline) ProcessEntry(db *pg.DB, entry *types.EntryLine) error {
	ts, level, location, msg, err := p.spdlog.Parse(entry.Message)
	if err != nil {
		// Ignore parse error
		log.Debug("Dropped entry: ", entry.Message, err)
		return nil
	}

	parts := p.msgSendParser.FindStringSubmatch(msg)
	if len(parts) == 3 {
		parts = parts[1:]
		msgid, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			// Ignore parse error
			log.Debug("Dropped entry: ", msg, err)
			return nil
		}
		obj := &MsgSend{
			entry.Host,
			entry.Inode,
			entry.Offset,
			ts,
			level,
			location,
			msgid,
			parts[1],
		}
		_, err = db.Model(obj).
			OnConflict("DO NOTHING").
			Insert()
		if err != nil {
			return err
		}
	}

	return nil
}
