package types

type EntryLine struct {
	Service string `pg:",unique:dedup"`
	Host    string `pg:",unique:dedup"`
	Inode   uint64  `pg:",unique:dedup"`
	Offset  uint64  `pg:",unique:dedup"`
	Message string
	Callback chan struct{}
}

type Service struct {
	Service    string
	LogRootDir string
	FileRegex  string
}
