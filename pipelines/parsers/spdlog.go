package parsers

import (
	"errors"
	"regexp"
	"time"
)

type SpdlogParser struct {
	parser *regexp.Regexp
	loc    *time.Location
}

// Log lines have the time without a timezone. spdlog writes local time of the
// process that logs unless it is set up otherwise, loc is the timezone to read it in.
func NewSpdlogParser(loc *time.Location) *SpdlogParser {
	re := regexp.MustCompile(`\[(.*)\] \[(.*)\] \[(.*)\] (.*)`)
	if loc == nil {
		loc = time.Local
	}
	return &SpdlogParser{re, loc}
}

func (p *SpdlogParser) Parse(line string) (ts time.Time, level string, location string, msg string, err error) {
	parts := p.parser.FindStringSubmatch(line)
	if len(parts) != 5 {
		err = errors.New("Parse error")
		return
	}

	parts = parts[1:]

	ts, err = time.ParseInLocation("2006-01-02 15:04:05.000", parts[0], p.loc)
	level = parts[1]
	location = parts[2]
	msg = parts[3]
	return
}
