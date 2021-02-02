package probe

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
	msgRecvParser *regexp.Regexp
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		parsers.NewSpdlogParser(),
		regexp.MustCompile("Msg log: ([0-9]+), cluster: (\\w+), relay: ([^\\s]+)"),
	}
}

type MsgRecv struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	MessageId uint64
	Cluster   string
	Relay     string
}

func (p *Pipeline) Setup(db *pg.DB) error {
	return pipelines.ApplyMigrations(db, "probe", probeMigrations[:], uint64(len(probeMigrations)))
}

func (p *Pipeline) ProcessEntry(db *pg.DB, entry *types.EntryLine) error {
	ts, level, location, msg, err := p.spdlog.Parse(entry.Message)
	if err != nil {
		// Ignore parse error
		log.Info("Dropped entry: ", entry.Message, err)
		return nil
	}

	parts := p.msgRecvParser.FindStringSubmatch(msg)
	if len(parts) == 4 {
		parts = parts[1:]
		msgid, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			// Ignore parse error
			log.Info("Dropped entry: ", msg, err)
			return nil
		}
		obj := &MsgRecv{
			entry.Host,
			entry.Inode,
			entry.Offset,
			ts,
			level,
			location,
			msgid,
			parts[1],
			parts[2],
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
