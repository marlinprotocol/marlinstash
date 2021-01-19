package types

type EntryLine struct {
	Service  string
	Host     string
	Inode    string
	Offset   string
	Message  string
	Callback chan EntryLine
}

type Service struct {
	Service    string
	LogRootDir string
	FileRegex  string
}
