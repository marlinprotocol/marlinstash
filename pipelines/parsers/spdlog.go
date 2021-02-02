package parsers

import (
	"errors"
	"regexp"
	"time"
)

type SpdlogParser struct {
	parser *regexp.Regexp
}

func NewSpdlogParser() *SpdlogParser {
	re := regexp.MustCompile(`\[(.*)\] \[(.*)\] \[(.*)\] (.*)`)
	return &SpdlogParser{re}
}

func (p *SpdlogParser) Parse(line string) (ts time.Time, level string, location string, msg string, err error) {
	parts := p.parser.FindStringSubmatch(line)
	if len(parts) != 5 {
		err = errors.New("Parse error")
		return
	}

	parts = parts[1:]

	ts, err = time.Parse("2006-01-02 15:04:05.000", parts[0])
	level = parts[1]
	location = parts[2]
	msg = parts[3]
	return
}
