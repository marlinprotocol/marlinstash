package types

type EntryLine struct {
	Service string `pg:",unique:dedup"`
	Host    string `pg:",unique:dedup"`
	Inode   uint64 `pg:",unique:dedup"`
	Offset  uint64 `pg:",unique:dedup"`
	Message string
}

type InodeOffset struct {
	Service string `pg:",unique:dedup"`
	Host    string `pg:",unique:dedup"`
	Inode   uint64 `pg:",unique:dedup"`
	Offset  uint64
}

type InodeOffsetReq struct {
	Service string
	Host    string
	Inode   uint64
	Resp    chan *InodeOffset
}

type Service struct {
	Service    string
	LogRootDir string
	FileRegex  string
}
