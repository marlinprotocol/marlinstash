package service

import (
	"fmt"
	"marlinstash/tail"
	"marlinstash/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	lf "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const REFRESH time.Duration = 30

func getPersistedOffset(inode uint64) uint64 {
	var offset uint64
	loc := viper.GetString("inode_info_directory") + "/" + strconv.FormatInt(int64(inode), 10)
	if _, err := os.Stat(loc); !os.IsNotExist(err) {
		fd, err := os.Open(loc)
		if err != nil {
			lf.Error("Error while reading "+loc, err)
		}
		defer fd.Close()
		_, err = fmt.Fscanf(fd, "%d\n", &offset)
		if err != nil {
			lf.Error("Error while reading "+loc, err)
		}
	}
	return offset
}

func setPersistedOffset(inode uint64, writerChan chan uint64) {
	loc := viper.GetString("inode_info_directory") + "/" + strconv.FormatInt(int64(inode), 10)
	c := 0
	gap := viper.GetInt("offset_persistence_gap")
	for offset := range writerChan {
		if c < gap {
			c = c + 1
			continue
		}
		fd, err := os.Open(loc)
		if err != nil {
			lf.Error("Error while opening "+loc, err)
		}
		_, err = fd.WriteString(fmt.Sprintf("%d\n", offset))
		if err != nil {
			lf.Error("Error while writing to "+loc, err)
		}
		defer fd.Close()
		c = 0
	}
}

func beginTail(service string, filepath string, offset uint64, datachan chan *types.EntryLine, restartSignal chan struct{}, inode uint64, writerChan chan uint64, log *lf.Entry) {
	log.Info("Tailer started for ", filepath, " persisted offset used: ", offset)
	t, _ := tail.TailFile(filepath, tail.Config{
		Location: &tail.SeekInfo{Offset: int64(offset)},
		Follow:   true,
	})
	host := viper.GetString("host")
	for line := range t.Lines {
		datachan <- &types.EntryLine{
			Service:  service,
			Host:     host,
			Inode:    inode,
			Offset:   offset + line.Offset,  // ROSHAN: Is this right?
			Message:  line.Text,
		}

		offset = offset + line.Offset

		writerChan <- offset
	}
}

func invokeTailer(service string, filepath string, datachan chan *types.EntryLine, log *lf.Entry) {
	log = log.WithField("file", filepath)
	fileinfo, _ := os.Stat(filepath)
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		log.Info("Not a syscall.Stat_t")
		return
	}
	inode := stat.Ino
	writerChan := make(chan uint64, 100)
	go setPersistedOffset(inode, writerChan)
	for {
		restartSignal := make(chan struct{})
		offset := getPersistedOffset(inode)
		go beginTail(service, filepath, offset, datachan, restartSignal, inode, writerChan, log)

		select {
		case <-restartSignal:
			continue
		}
	}
}

func Run(t types.Service, datachan chan *types.EntryLine) {
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
						go invokeTailer(t.Service, t.LogRootDir+"/"+f.Name(), datachan, log)
						invokedRoutines[f.Name()] = true
					}
				}
			}
			return nil
		})
		time.Sleep(REFRESH * time.Second)
	}
}
