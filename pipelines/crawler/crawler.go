package crawler

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
	clusterParser *regexp.Regexp
	relayParser   *regexp.Regexp
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		parsers.NewSpdlogParser(),
		regexp.MustCompile("New cluster: (0x[[:xdigit:]]{40}), (\\d+\\.\\d+\\.\\d+\\.\\d+:\\d+)"),
		regexp.MustCompile("New peer: (0x[[:xdigit:]]{40}), (\\d+\\.\\d+\\.\\d+\\.\\d+:\\d+)"),
	}
}

type Cluster struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	Ip        string
	Cluster   string
}

type ArchivedCluster struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	Ip        string
	Cluster   string
}

type Relay struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	Ip        string
	Cluster   string
}

type ArchivedRelay struct {
	Host      string
	Inode     uint64
	Offset    uint64
	Ts        time.Time
	Level     string
	Location  string
	Ip        string
	Cluster   string
}


func (p *Pipeline) Setup(db *pg.DB) error {
	return pipelines.ApplyMigrations(db, "crawler", crawlerMigrations[:], uint64(len(crawlerMigrations)))
}

func (p *Pipeline) ProcessEntry(db *pg.DB, entry *types.EntryLine) error {
	ts, level, location, msg, err := p.spdlog.Parse(entry.Message)
	if err != nil {
		// Ignore parse error
		log.Debug("Dropped entry: ", entry.Message, err)
		return nil
	}

	parts := p.clusterParser.FindStringSubmatch(msg)
	if len(parts) == 3 {
		parts = parts[1:]
		if err != nil {
			// Ignore parse error
			log.Debug("Dropped entry: ", msg, err)
			return nil
		}
		obj := &Cluster{
			entry.Host,
			entry.Inode,
			entry.Offset,
			ts,
			level,
			location,
			parts[1],
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

	parts = p.relayParser.FindStringSubmatch(msg)
	if len(parts) == 3 {
		parts = parts[1:]
		if err != nil {
			// Ignore parse error
			log.Debug("Dropped entry: ", msg, err)
			return nil
		}
		obj := &Relay{
			entry.Host,
			entry.Inode,
			entry.Offset,
			ts,
			level,
			location,
			parts[1],
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
	_, err := tx.Query(&ArchivedCluster{}, "INSERT INTO archived_clusters SELECT * FROM clusters WHERE host = ? AND inode = ?;", host, inode)
	if err != nil {
		return err
	}

	_, err = tx.Query(&ArchivedCluster{}, "DELETE FROM clusters WHERE host = ? AND inode = ?;", host, inode)
	if err != nil {
		return err
	}

	_, err = tx.Query(&ArchivedRelay{}, "INSERT INTO archived_relays SELECT * FROM relays WHERE host = ? AND inode = ?;", host, inode)
	if err != nil {
		return err
	}

	_, err = tx.Query(&ArchivedRelay{}, "DELETE FROM relays WHERE host = ? AND inode = ?;", host, inode)
	if err != nil {
		return err
	}

	return nil
}
