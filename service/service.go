package service

import (
	"marlinstash/tail"
	"marlinstash/types"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	lf "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const REFRESH time.Duration = 30

func beginTail(service string, host string, filepath string, offset uint64, datachan chan *types.EntryLine, restartSignal chan struct{}, inode uint64, log *lf.Entry) {
	log.Info("Tailer started for ", filepath, " persisted offset used: ", offset)
	t, _ := tail.TailFile(filepath, tail.Config{
		Location: &tail.SeekInfo{Offset: int64(offset)},
		Follow:   true,
	})
	for line := range t.Lines {
		datachan <- &types.EntryLine{
			Service: service,
			Host:    host,
			Inode:   inode,
			Offset:  line.Offset,
			Message: line.Text,
		}
	}
}

func invokeTailer(service string, filepath string, datachan chan *types.EntryLine, inodeOffsetReqChan chan *types.InodeOffsetReq, log *lf.Entry) {
	log = log.WithField("file", filepath)
	fileinfo, _ := os.Stat(filepath)
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		log.Info("Not a syscall.Stat_t")
		return
	}
	inode := stat.Ino
	for {
		restartSignal := make(chan struct{}) // TODO: not used right now
		respChan := make(chan *types.InodeOffset)
		host := viper.GetString("host")
		inodeOffsetReqChan <- &types.InodeOffsetReq{
			Service: service,
			Host:    host,
			Inode:   inode,
			Resp:    respChan,
		}
		offsetResponse := <-respChan
		go beginTail(service, host, filepath, offsetResponse.Offset, datachan, restartSignal, inode, log)

		select {
		case <-restartSignal:
			continue
		}
	}
}

func Run(t types.Service, datachan chan *types.EntryLine, inodeOffsetReqChan chan *types.InodeOffsetReq) {
	log := lf.WithField("Service", t.Service)
	log.Info("Service start for "+t.Service+" with regex: ", t.FileRegex)

	invokedRoutines := make(map[string]bool)

	for {
		log.Info("Checking for new files to tail if any")
		_ = filepath.Walk(t.LogRootDir, func(path string, f os.FileInfo, _ error) error {
			if !f.IsDir() {
				r, err := regexp.MatchString(t.FileRegex, f.Name())
				if err == nil && r {
					if _, ok := invokedRoutines[f.Name()]; !ok {
						go invokeTailer(t.Service, t.LogRootDir+"/"+f.Name(), datachan, inodeOffsetReqChan, log)
						invokedRoutines[f.Name()] = true
					}
				}
			}
			return nil
		})
		time.Sleep(REFRESH * time.Second)
	}
}
