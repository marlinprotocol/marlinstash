package service

import (
	"errors"
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

func beginTail(service string, host string, filepath string, offset uint64, datachan chan *types.EntryLine, killSignal chan uint64, inode uint64, log *lf.Entry) {
	log.WithField("offset", offset).Info("Tailer started")
	t, _ := tail.TailFile(filepath, tail.Config{
		Location: &tail.SeekInfo{Offset: int64(offset)},
		Follow:   true,
	})
	for line := range t.Lines {
		if line.Err != nil {
			log.Info("Tailer stopped due to ", line.Err)
			killSignal <- inode
			return
		}
		datachan <- &types.EntryLine{
			Service: service,
			Host:    host,
			Inode:   inode,
			Offset:  line.Offset,
			Message: line.Text,
		}
	}
}

func Run(t types.Service, datachan chan *types.EntryLine, inodeOffsetReqChan, resetReqChan chan *types.InodeOffsetReq) {
	log := lf.WithField("service", t.Service)
	log.WithField("regex", t.FileRegex).WithField("refresh", REFRESH*time.Second).Info("Service started")

	invokedRoutines := make(map[uint64]bool)
	host := viper.GetString("host")
	killSignal := make(chan uint64)

	for {
		log.Info("Checking for new files to tail")
		_ = filepath.Walk(t.LogRootDir, func(path string, f os.FileInfo, _ error) error {
			if !f.IsDir() {
				r, err := regexp.MatchString(t.FileRegex, f.Name())
				if err == nil && r {
					filepath := t.LogRootDir + "/" + f.Name()
					fileinfo, _ := os.Stat(filepath)
					stat, ok := fileinfo.Sys().(*syscall.Stat_t)
					if !ok {
						log.Info("Not a syscall.Stat_t")
						return errors.New("Not a syscall.Stat_t")
					}
					inode := stat.Ino
					if isCurrentlyRunning, ok := invokedRoutines[inode]; !ok || !isCurrentlyRunning {
						if !ok {
							log.Info("Running inode tailer for the first time for inode: ", inode)
						} else {
							log.Info("Seems like inode tailer is reinvoked for same inode again: ", inode)
						}
						respChan := make(chan *types.InodeOffset)
						inodeOffsetReqChan <- &types.InodeOffsetReq{
							Service: t.Service,
							Host:    host,
							Inode:   inode,
							Resp:    respChan,
						}
						log.Info("Waiting for DB to return offset for ", inode)
						offsetResponse := <-respChan
						log.Info("DB responded with offset", offsetResponse, " for inode ", inode)

						// Mitigate inode reuse
						fi, err := os.Stat(filepath)
						if err != nil {
							return err
						}

						// Check size
						size := fi.Size()
						if uint64(size) < offsetResponse.Offset/10 {
							log.Info("Found large size difference between current file size and offset stored in DB for inode: ", inode)
							respChan := make(chan *types.InodeOffset)
							resetReqChan <- &types.InodeOffsetReq{
								Service: t.Service,
								Host:    host,
								Inode:   inode,
								Resp:    respChan,
							}
							_, ok := <-respChan
							if !ok {
								return errors.New("Reset failure")
							}
							offsetResponse.Offset = 0
						}

						go beginTail(t.Service, host, filepath, offsetResponse.Offset, datachan, killSignal, inode, log.WithField("file", filepath))
						// go invokeTailer(t.Service, t.LogRootDir+"/"+f.Name(), datachan, inodeOffsetReqChan, log)
						invokedRoutines[inode] = true
					}
				}
			}
			return nil
		})

	WAIT:
		for {
			select {
			case inodeTailerKilled := <-killSignal:
				log.Info("Tailer has been killed for: ", inodeTailerKilled)
				invokedRoutines[inodeTailerKilled] = false
			case <-time.After(REFRESH * time.Second):
				// REDO FILEWALK HERE
				break WAIT
			}
		}
	}
}
