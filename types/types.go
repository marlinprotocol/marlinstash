package types

type EntryLine struct {
	Service  string
	Host     string
	Inode    string
	Offset   int64
	Message  string
	Callback chan struct{}
}

type Service struct {
	Service    string
	LogRootDir string
	FileRegex  string
}
