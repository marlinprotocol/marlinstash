package types

import "time"

type DBEntryLine struct {
	Service   string
	Inode     string
	Timestamp time.Time
	FileName  string
	Offset    string
	Message   string
}
